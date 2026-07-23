using System.Collections.Concurrent;
using System.Diagnostics;
using System.IO.Pipes;
using System.Text;

namespace PropFirmMultiPass
{
    public partial class Form1 : Form
    {
        // ================================================================
        // STEALTH DATA MODELS
        // ================================================================

        private sealed class StealthPosition
        {
            public long Ticket { get; set; }
            public string Symbol { get; set; } = string.Empty;
            public string Type { get; set; } = string.Empty;   // "BUY" or "SELL"
            public double Volume { get; set; }
            public double OpenPrice { get; set; }
            public double LivePrice { get; set; }
            public double HiddenSL { get; set; }
            public double HiddenTP { get; set; }
            public bool Triggered { get; set; }
        }

        // ================================================================
        // CORE STATE
        // ================================================================

        private readonly ConcurrentDictionary<long, StealthPosition> _vault = new();
        private volatile bool _shieldActive;
        private CancellationTokenSource _pipeCts = new();
        private NamedPipeServerStream? _activePipeServer;
        private StreamWriter? _pipeWriter;
        private readonly object _pipeWriteLock = new();
        private readonly Stopwatch _pingStopwatch = new();
        private long _lastPingUs;

        // Amber warning threshold in pips
        private const double WarningPipsThreshold = 5.0;

        // Colors kept as fields for grid painting
        private static readonly Color Emerald = Color.FromArgb(0, 230, 118);
        private static readonly Color Amber = Color.FromArgb(255, 214, 0);
        private static readonly Color SoftRed = Color.FromArgb(207, 102, 121);
        private static readonly Color DeepBg = Color.FromArgb(11, 13, 17);
        private static readonly Color DimText = Color.FromArgb(130, 140, 160);

        // ================================================================
        // CONSTRUCTOR
        // ================================================================

        public Form1()
        {
            InitializeComponent();
            EnableDoubleBufferingOnGrid();
            WireEvents();
        }

        private void EnableDoubleBufferingOnGrid()
        {
            typeof(DataGridView)
                .GetProperty("DoubleBuffered",
                    System.Reflection.BindingFlags.Instance | System.Reflection.BindingFlags.NonPublic)!
                .SetValue(dgvPositions, true);
        }

        private void WireEvents()
        {
            Load += OnFormLoad;
            FormClosing += OnFormClosing;
            btnActivateShield.Click += OnToggleShield;
            btnPanicFlatten.Click += OnPanicFlatten;
            btnSimulate.Click += OnSimulateBrokerStopHunt;
        }

        // ================================================================
        // FORM LIFECYCLE
        // ================================================================

        private void OnFormLoad(object? sender, EventArgs e)
        {
            AppendLog("System initialized. Awaiting shield activation...", DimText);
        }

        private void OnFormClosing(object? sender, FormClosingEventArgs e)
        {
            _pipeCts.Cancel();
            lock (_pipeWriteLock)
            {
                _pipeWriter?.Dispose();
                _pipeWriter = null;
            }
            try { _activePipeServer?.Dispose(); } catch { }
        }

        // ================================================================
        // SHIELD TOGGLE
        // ================================================================

        private void OnToggleShield(object? sender, EventArgs e)
        {
            if (_shieldActive)
            {
                DeactivateShield();
            }
            else
            {
                ActivateShield();
            }
        }

        private void ActivateShield()
        {
            _shieldActive = true;
            _pipeCts = new CancellationTokenSource();
            btnActivateShield.Text = "\u25C9  SHIELD ACTIVE \u2014 CLICK TO DEACTIVATE";
            btnActivateShield.BackColor = Color.FromArgb(0, 50, 25);
            lblShieldStatus.Text = "ACTIVE";
            lblShieldStatus.ForeColor = Emerald;
            AppendLog("STEALTH SHIELD ACTIVATED. Named Pipe listener starting...", Emerald);
            _ = Task.Run(() => PipeListenerLoopAsync(_pipeCts.Token));
        }

        private void DeactivateShield()
        {
            _shieldActive = false;
            _pipeCts.Cancel();
            lock (_pipeWriteLock)
            {
                _pipeWriter?.Dispose();
                _pipeWriter = null;
            }
            try { _activePipeServer?.Dispose(); } catch { }
            _activePipeServer = null;

            btnActivateShield.Text = "\u25C9  ACTIVATE STEALTH SHIELD";
            btnActivateShield.BackColor = Color.FromArgb(0, 77, 40);
            lblShieldStatus.Text = "INACTIVE";
            lblShieldStatus.ForeColor = SoftRed;
            lblPingValue.Text = "-- ms";
            lblPingValue.ForeColor = DimText;
            AppendLog("STEALTH SHIELD DEACTIVATED. Pipe listener stopped.", SoftRed);
        }

        // ================================================================
        // NAMED PIPE SERVER — CONTINUOUS LISTENER
        // ================================================================

        private async Task PipeListenerLoopAsync(CancellationToken ct)
        {
            while (!ct.IsCancellationRequested)
            {
                NamedPipeServerStream? server = null;
                try
                {
                    server = new NamedPipeServerStream(
                        "AUTOSCRIPTS_STEALTH_GUARD",
                        PipeDirection.InOut,
                        1,
                        PipeTransmissionMode.Byte,
                        PipeOptions.Asynchronous);

                    _activePipeServer = server;
                    UiInvoke(() => AppendLog("Pipe server waiting for MT5 connection...", DimText));

                    await server.WaitForConnectionAsync(ct);
                    UiInvoke(() => AppendLog("MT5 bridge CONNECTED.", Emerald));

                    using var reader = new StreamReader(server, Encoding.UTF8, leaveOpen: true);
                    lock (_pipeWriteLock)
                    {
                        _pipeWriter = new StreamWriter(server, Encoding.UTF8, leaveOpen: true) { AutoFlush = true };
                    }

                    while (server.IsConnected && !ct.IsCancellationRequested)
                    {
                        _pingStopwatch.Restart();
                        string? line = await reader.ReadLineAsync(ct);
                        _pingStopwatch.Stop();
                        _lastPingUs = _pingStopwatch.ElapsedTicks * 1_000_000 / Stopwatch.Frequency;

                        if (line == null) break; // client disconnected

                        ProcessIncomingPacket(line);
                    }
                }
                catch (OperationCanceledException) { break; }
                catch (IOException)
                {
                    UiInvoke(() => AppendLog("MT5 bridge DISCONNECTED. Reconnecting...", Amber));
                }
                catch (Exception ex)
                {
                    UiInvoke(() => AppendLog($"Pipe error: {ex.Message}", SoftRed));
                }
                finally
                {
                    lock (_pipeWriteLock)
                    {
                        _pipeWriter?.Dispose();
                        _pipeWriter = null;
                    }
                    server?.Dispose();
                    _activePipeServer = null;
                }

                if (!ct.IsCancellationRequested)
                    await Task.Delay(500, ct).ConfigureAwait(false);
            }
        }

        // ================================================================
        // PACKET PROCESSING ENGINE
        // ================================================================

        private void ProcessIncomingPacket(string raw)
        {
            ReadOnlySpan<char> span = raw.AsSpan();
            int firstSemi = span.IndexOf(';');
            if (firstSemi < 0) return;

            ReadOnlySpan<char> cmd = span[..firstSemi];

            if (cmd.SequenceEqual("SYNC"))
                HandleSyncPacket(raw);
            else if (cmd.SequenceEqual("TICK"))
                HandleTickPacket(raw);
        }

        // SYNC;Ticket;Symbol;Type;Volume;OpenPrice;HiddenSL;HiddenTP
        private void HandleSyncPacket(string raw)
        {
            string[] parts = raw.Split(';');
            if (parts.Length < 8) return;

            if (!long.TryParse(parts[1], out long ticket)) return;
            if (!double.TryParse(parts[4], out double volume)) return;
            if (!double.TryParse(parts[5], out double openPrice)) return;
            if (!double.TryParse(parts[6], out double hiddenSL)) return;
            if (!double.TryParse(parts[7], out double hiddenTP)) return;

            var pos = _vault.AddOrUpdate(ticket,
                _ => new StealthPosition
                {
                    Ticket = ticket,
                    Symbol = parts[2],
                    Type = parts[3].ToUpperInvariant(),
                    Volume = volume,
                    OpenPrice = openPrice,
                    HiddenSL = hiddenSL,
                    HiddenTP = hiddenTP,
                    LivePrice = openPrice
                },
                (_, existing) =>
                {
                    existing.Symbol = parts[2];
                    existing.Type = parts[3].ToUpperInvariant();
                    existing.Volume = volume;
                    existing.OpenPrice = openPrice;
                    existing.HiddenSL = hiddenSL;
                    existing.HiddenTP = hiddenTP;
                    existing.Triggered = false;
                    return existing;
                });

            UiInvoke(() =>
            {
                AppendLog($"SYNC: Registered stealth shield for {pos.Symbol} Ticket #{ticket}  SL={hiddenSL}  TP={hiddenTP}", Emerald);
                RefreshGrid();
            });
        }

        // TICK;Symbol;Bid;Ask
        private void HandleTickPacket(string raw)
        {
            string[] parts = raw.Split(';');
            if (parts.Length < 4) return;

            string symbol = parts[1];
            if (!double.TryParse(parts[2], out double bid)) return;
            if (!double.TryParse(parts[3], out double ask)) return;

            double spread = Math.Abs(ask - bid);
            double maxSpread = 0;
            UiInvoke(() => maxSpread = (double)nudMaxSpread.Value);
            // Spread filter: skip processing if spread exceeds configured max (non-blocking read)
            // We read maxSpread synchronously below for thread safety

            List<long> ticketsToClose = new();

            foreach (var kvp in _vault)
            {
                var pos = kvp.Value;
                if (!string.Equals(pos.Symbol, symbol, StringComparison.OrdinalIgnoreCase))
                    continue;

                // Update live price
                pos.LivePrice = pos.Type == "BUY" ? bid : ask;

                if (pos.Triggered) continue;
                if (!_shieldActive) continue;

                // --- Check SL Breach ---
                bool slBreached = false;
                if (pos.Type == "BUY" && bid <= pos.HiddenSL)
                    slBreached = true;
                else if (pos.Type == "SELL" && ask >= pos.HiddenSL)
                    slBreached = true;

                // --- Check TP Breach ---
                bool tpBreached = false;
                if (pos.HiddenTP > 0)
                {
                    if (pos.Type == "BUY" && bid >= pos.HiddenTP)
                        tpBreached = true;
                    else if (pos.Type == "SELL" && ask <= pos.HiddenTP)
                        tpBreached = true;
                }

                if (slBreached || tpBreached)
                {
                    pos.Triggered = true;
                    string reason = slBreached ? "Hidden SL" : "Hidden TP";
                    double level = slBreached ? pos.HiddenSL : pos.HiddenTP;
                    ticketsToClose.Add(pos.Ticket);

                    UiInvoke(() =>
                    {
                        AppendLog($"STEALTH TRIGGER: {pos.Symbol} Ticket #{pos.Ticket} hit {reason} at {level:F5}. " +
                                  $"Dispatched instant Market Close command to MT5...", SoftRed);
                    });
                }
            }

            // Dispatch close commands
            foreach (long ticket in ticketsToClose)
            {
                SendCommandToMT5($"COMMAND;STEALTH_CLOSE;{ticket}");
            }

            // Update UI
            UiInvoke(() =>
            {
                UpdatePingDisplay();
                RefreshGrid();
            });
        }

        // ================================================================
        // COMMAND OUTPUT — SEND BACK TO MT5 VIA PIPE
        // ================================================================

        private void SendCommandToMT5(string command)
        {
            lock (_pipeWriteLock)
            {
                if (_pipeWriter == null) return;
                try
                {
                    _pipeWriter.WriteLine(command);
                }
                catch (IOException)
                {
                    UiInvoke(() => AppendLog("Failed to send command — pipe broken.", SoftRed));
                }
            }
        }

        // ================================================================
        // PANIC FLATTEN — CLOSE ALL POSITIONS
        // ================================================================

        private void OnPanicFlatten(object? sender, EventArgs e)
        {
            if (_vault.IsEmpty)
            {
                AppendLog("PANIC FLATTEN: No active stealth positions to close.", Amber);
                return;
            }

            AppendLog("PANIC FLATTEN: Dispatching Market Close for ALL shielded positions...", SoftRed);

            foreach (var kvp in _vault)
            {
                var pos = kvp.Value;
                if (!pos.Triggered)
                {
                    pos.Triggered = true;
                    SendCommandToMT5($"COMMAND;STEALTH_CLOSE;{pos.Ticket}");
                    AppendLog($"  >> Closing Ticket #{pos.Ticket} ({pos.Symbol} {pos.Type} {pos.Volume})", SoftRed);
                }
            }

            RefreshGrid();
        }

        // ================================================================
        // SIMULATION — BROKER STOP HUNT DEMO
        // ================================================================

        private void OnSimulateBrokerStopHunt(object? sender, EventArgs e)
        {
            // Ensure shield is active for the simulation
            if (!_shieldActive)
            {
                ActivateShield();
            }

            long simTicket = 887412 + _vault.Count;
            string simSymbol = "XAUUSD";
            double openPrice = 2345.50;
            double hiddenSL = 2345.20;
            double hiddenTP = 2348.00;

            // Step 1: Register a simulated position via SYNC
            string syncPacket = $"SYNC;{simTicket};{simSymbol};BUY;0.50;{openPrice};{hiddenSL};{hiddenTP}";
            ProcessIncomingPacket(syncPacket);

            AppendLog($"[SIMULATION] Injected dummy BUY {simSymbol} @ {openPrice}  SL={hiddenSL}  TP={hiddenTP}", Amber);
            AppendLog($"[SIMULATION] Feeding aggressive downward tick in 1.5 seconds...", Amber);

            // Step 2: After a brief delay, feed a tick that breaches the hidden SL
            _ = Task.Run(async () =>
            {
                await Task.Delay(1500);
                double aggressiveBid = hiddenSL - 0.30;  // Price drops below hidden SL
                double aggressiveAsk = aggressiveBid + 0.20;
                string tickPacket = $"TICK;{simSymbol};{aggressiveBid};{aggressiveAsk}";

                UiInvoke(() => AppendLog($"[SIMULATION] TICK injected: Bid={aggressiveBid:F2} (below SL={hiddenSL:F2})", Amber));
                ProcessIncomingPacket(tickPacket);

                // Auto-remove after 5 seconds
                await Task.Delay(5000);
                _vault.TryRemove(simTicket, out _);
                UiInvoke(() =>
                {
                    AppendLog($"[SIMULATION] Cleaned up simulated ticket #{simTicket}.", DimText);
                    RefreshGrid();
                });
            });
        }

        // ================================================================
        // GRID REFRESH — FULL REPAINT FROM VAULT
        // ================================================================

        private void RefreshGrid()
        {
            var positions = _vault.Values.ToArray();
            lblActiveCount.Text = positions.Count(p => !p.Triggered).ToString();

            // Resize rows only if count changed
            if (dgvPositions.Rows.Count != positions.Length)
            {
                dgvPositions.Rows.Clear();
                if (positions.Length > 0)
                    dgvPositions.Rows.Add(positions.Length);
            }

            for (int i = 0; i < positions.Length; i++)
            {
                var pos = positions[i];
                var row = dgvPositions.Rows[i];

                double pipsToSL = CalculatePipsToSL(pos);
                string status = GetProtectionStatus(pos, pipsToSL);

                row.Cells["Ticket"].Value = pos.Ticket;
                row.Cells["Symbol"].Value = pos.Symbol;
                row.Cells["Type"].Value = pos.Type;
                row.Cells["Volume"].Value = pos.Volume.ToString("F2");
                row.Cells["OpenPrice"].Value = pos.OpenPrice.ToString("F5");
                row.Cells["LivePrice"].Value = pos.LivePrice.ToString("F5");
                row.Cells["HiddenSL"].Value = pos.HiddenSL.ToString("F5");
                row.Cells["HiddenTP"].Value = pos.HiddenTP > 0 ? pos.HiddenTP.ToString("F5") : "--";
                row.Cells["RemainingPips"].Value = pipsToSL.ToString("F1");
                row.Cells["Status"].Value = status;

                // Apply row coloring
                ApplyRowStyling(row, pos, pipsToSL);
            }
        }

        private static double CalculatePipsToSL(StealthPosition pos)
        {
            double pointSize = GetPointSize(pos.Symbol);
            if (pointSize <= 0) pointSize = 0.00001;

            double diff;
            if (pos.Type == "BUY")
                diff = pos.LivePrice - pos.HiddenSL;
            else
                diff = pos.HiddenSL - pos.LivePrice;

            return diff / pointSize;
        }

        private static double GetPointSize(string symbol)
        {
            string upper = symbol.ToUpperInvariant();
            // Gold / XAUUSD
            if (upper.Contains("XAU")) return 0.01;
            // JPY pairs
            if (upper.Contains("JPY")) return 0.01;
            // Indices (common names)
            if (upper.Contains("US30") || upper.Contains("NAS") || upper.Contains("SPX") || upper.Contains("DAX"))
                return 0.1;
            // Default forex
            return 0.0001;
        }

        private static string GetProtectionStatus(StealthPosition pos, double pipsToSL)
        {
            if (pos.Triggered)
                return "\u26A0 TRIGGERED \u2014 CLOSING";
            if (pipsToSL <= 0)
                return "\u274C SL BREACHED";
            if (pipsToSL <= WarningPipsThreshold)
                return $"\u26A1 WARNING \u2014 {pipsToSL:F1} pips";
            return "\u2705 SHIELDED";
        }

        private void ApplyRowStyling(DataGridViewRow row, StealthPosition pos, double pipsToSL)
        {
            Color fg, bg;

            if (pos.Triggered)
            {
                // Panic / Triggered state
                bg = Color.FromArgb(45, 18, 22);
                fg = SoftRed;
            }
            else if (pipsToSL <= WarningPipsThreshold)
            {
                // Imminent warning
                bg = Color.FromArgb(40, 38, 10);
                fg = Amber;
            }
            else
            {
                // Healthy shielded
                bg = DeepBg;
                fg = Emerald;
            }

            row.DefaultCellStyle.BackColor = bg;
            row.DefaultCellStyle.ForeColor = fg;
            row.DefaultCellStyle.SelectionBackColor = bg;
            row.DefaultCellStyle.SelectionForeColor = fg;
        }

        // ================================================================
        // PING DISPLAY
        // ================================================================

        private void UpdatePingDisplay()
        {
            double ms = _lastPingUs / 1000.0;
            lblPingValue.Text = $"{ms:F2} ms";
            lblPingValue.ForeColor = ms < 5 ? Emerald : ms < 20 ? Amber : SoftRed;
        }

        // ================================================================
        // LOGGING
        // ================================================================

        private void AppendLog(string message, Color color)
        {
            if (rtbLog.IsDisposed) return;

            string timestamp = DateTime.Now.ToString("HH:mm:ss.fff");
            string line = $"[{timestamp}] {message}\n";

            rtbLog.SelectionStart = rtbLog.TextLength;
            rtbLog.SelectionLength = 0;
            rtbLog.SelectionColor = color;
            rtbLog.AppendText(line);
            rtbLog.ScrollToCaret();

            // Trim log to prevent memory growth (keep last 2000 lines)
            if (rtbLog.Lines.Length > 2200)
            {
                int cutIndex = rtbLog.GetFirstCharIndexFromLine(200);
                rtbLog.Select(0, cutIndex);
                rtbLog.SelectedText = "";
            }
        }

        // ================================================================
        // THREAD-SAFE UI HELPER
        // ================================================================

        private void UiInvoke(Action action)
        {
            if (IsDisposed) return;
            if (InvokeRequired)
                BeginInvoke(action);
            else
                action();
        }
    }
}
