using System;
using System.Globalization;
using System.IO;
using System.IO.Pipes;
using System.Text;
using System.Threading;
using System.Threading.Tasks;
using System.Windows.Forms;

namespace PropFirmDrawdown
{
    /// <summary>
    /// Advanced Prop-Firm Drawdown & Risk Guard.
    /// Hosts an async NamedPipeServerStream that consumes METRICS packets from an
    /// MT5 EA, evaluates strict risk rules, and broadcasts a KILL_SWITCH command
    /// when a hard limit is breached.
    /// </summary>
    public partial class Form1 : Form
    {
        private const string PipeName = "AUTOSCRIPTS_RISK_GUARD";
        private const string KillCommand = "COMMAND;KILL_SWITCH;CLOSE_ALL_AND_DISABLE\n";

        private CancellationTokenSource? _cts;
        private Task? _pipeLoopTask;
        private volatile bool _pipeRunning;

        // Active connection state (written/read across threads; guarded by _streamLock).
        private readonly object _streamLock = new();
        private NamedPipeServerStream? _activeStream;
        private StreamWriter? _activeWriter;

        // Active strict rules (snapshotted off the UI on Apply).
        private readonly object _rulesLock = new();
        private decimal _maxDailyDdPct = 5M;
        private decimal _maxAbsoluteDd = 1000M;
        private decimal _maxLotSize = 1.00M;
        private int _maxOpenPositions = 5;

        // Lockout latch — once tripped, only an app restart clears it.
        private volatile bool _isLocked;

        // Last-known metrics for UI consistency.
        private AccountMetrics _lastMetrics;

        public Form1()
        {
            InitializeComponent();
        }

        // ============================================================
        // Lifecycle
        // ============================================================
        private void Form1_Load(object? sender, EventArgs e)
        {
            SnapshotRulesFromUi();
            AppendLog("Risk Guard initialized. Press CONNECT TO MT5 to start the pipe listener.", LogLevel.Info);
            SetConnectButtonState(false);
        }

        private void Form1_FormClosing(object? sender, FormClosingEventArgs e)
        {
            StopPipeServer(joinTimeoutMs: 500);
        }

        // ============================================================
        // Connect / Disconnect (manual MT5 link toggle)
        // ============================================================
        private void BtnConnect_Click(object? sender, EventArgs e)
        {
            if (_isLocked)
            {
                AppendLog("Connection toggle rejected — guard is LOCKED.", LogLevel.Warn);
                return;
            }

            if (_pipeRunning)
                StopPipeServer(joinTimeoutMs: 1000);
            else
                StartPipeServer();
        }

        private void StartPipeServer()
        {
            if (_pipeRunning) return;
            _cts = new CancellationTokenSource();
            _pipeRunning = true;
            SetConnectButtonState(true);
            AppendLog("Starting pipe listener on \\\\.\\pipe\\" + PipeName, LogLevel.Net);
            var token = _cts.Token;
            _pipeLoopTask = Task.Run(() => PipeServerLoopAsync(token));
        }

        private void StopPipeServer(int joinTimeoutMs)
        {
            if (!_pipeRunning && _cts == null) return;
            try { _cts?.Cancel(); } catch { }

            // Forcibly close any active client to unblock pending I/O.
            lock (_streamLock)
            {
                try { _activeWriter?.Dispose(); } catch { }
                try { _activeStream?.Dispose(); } catch { }
                _activeWriter = null;
                _activeStream = null;
            }

            try { _pipeLoopTask?.Wait(joinTimeoutMs); } catch { }

            _pipeRunning = false;
            _cts?.Dispose();
            _cts = null;
            _pipeLoopTask = null;

            SetConnectButtonState(false);
            SetConnectionStatus("DISCONNECTED", isConnected: false);
            AppendLog("Pipe listener stopped.", LogLevel.Warn);
        }

        private void SetConnectButtonState(bool running)
        {
            UiInvoke(() =>
            {
                if (running)
                {
                    btnConnect.Text = "DISCONNECT";
                    btnConnect.BackColor = System.Drawing.ColorTranslator.FromHtml("#FF8A6B");
                    btnConnect.ForeColor = System.Drawing.Color.Black;
                }
                else
                {
                    btnConnect.Text = "CONNECT TO MT5";
                    btnConnect.BackColor = System.Drawing.ColorTranslator.FromHtml("#00E6C3");
                    btnConnect.ForeColor = System.Drawing.Color.Black;
                }
            });
        }

        // ============================================================
        // UI marshaling helper
        // ============================================================
        private void UiInvoke(Action action)
        {
            if (IsDisposed || Disposing) return;
            if (InvokeRequired)
            {
                try { BeginInvoke(action); } catch { /* form closing */ }
            }
            else
            {
                action();
            }
        }

        // ============================================================
        // Logging
        // ============================================================
        private enum LogLevel { Info, Warn, Error, Breach, Net }

        private void AppendLog(string message, LogLevel level)
        {
            UiInvoke(() =>
            {
                var color = level switch
                {
                    LogLevel.Warn => System.Drawing.ColorTranslator.FromHtml("#F5C451"),
                    LogLevel.Error => System.Drawing.ColorTranslator.FromHtml("#FF8A6B"),
                    LogLevel.Breach => System.Drawing.ColorTranslator.FromHtml("#FF3B30"),
                    LogLevel.Net => System.Drawing.ColorTranslator.FromHtml("#00E6C3"),
                    _ => System.Drawing.ColorTranslator.FromHtml("#E6EAF2")
                };

                var stamp = DateTime.Now.ToString("HH:mm:ss.fff", CultureInfo.InvariantCulture);
                var tag = level.ToString().ToUpperInvariant().PadRight(6);

                rtbLog.SelectionStart = rtbLog.TextLength;
                rtbLog.SelectionLength = 0;
                rtbLog.SelectionColor = System.Drawing.ColorTranslator.FromHtml("#6B7488");
                rtbLog.AppendText($"[{stamp}] ");
                rtbLog.SelectionStart = rtbLog.TextLength;
                rtbLog.SelectionLength = 0;
                rtbLog.SelectionColor = color;
                rtbLog.AppendText($"{tag} ");
                rtbLog.SelectionColor = System.Drawing.ColorTranslator.FromHtml("#E6EAF2");
                rtbLog.AppendText(message + Environment.NewLine);
                rtbLog.SelectionStart = rtbLog.TextLength;
                rtbLog.ScrollToCaret();
            });
        }

        // ============================================================
        // Rules
        // ============================================================
        private void BtnApplyRules_Click(object? sender, EventArgs e)
        {
            if (_isLocked)
            {
                AppendLog("Rule change rejected — guard is LOCKED.", LogLevel.Warn);
                return;
            }
            SnapshotRulesFromUi();
            AppendLog(
                $"Strict rules applied: MaxDailyDD={_maxDailyDdPct}%  MaxAbsDD=${_maxAbsoluteDd}  MaxLot={_maxLotSize}  MaxPos={_maxOpenPositions}",
                LogLevel.Info);
        }

        private void SnapshotRulesFromUi()
        {
            lock (_rulesLock)
            {
                _maxDailyDdPct = numMaxDdPct.Value;
                _maxAbsoluteDd = numMaxAbsDd.Value;
                _maxLotSize = numMaxLot.Value;
                _maxOpenPositions = (int)numMaxPos.Value;
            }
        }

        // ============================================================
        // Manual kill switch
        // ============================================================
        private void BtnKillSwitch_Click(object? sender, EventArgs e)
        {
            AppendLog("MANUAL KILL SWITCH engaged by operator.", LogLevel.Breach);
            TriggerBreach("MANUAL_OVERRIDE");
        }

        // ============================================================
        // Pipe server loop
        // ============================================================
        private async Task PipeServerLoopAsync(CancellationToken ct)
        {
            while (!ct.IsCancellationRequested)
            {
                NamedPipeServerStream? server = null;
                try
                {
                    server = new NamedPipeServerStream(
                        PipeName,
                        PipeDirection.InOut,
                        1,
                        PipeTransmissionMode.Byte,
                        PipeOptions.Asynchronous);

                    SetConnectionStatus("AWAITING MT5...", isConnected: false);
                    AppendLog("Pipe server up — awaiting MT5 EA connection.", LogLevel.Net);

                    await server.WaitForConnectionAsync(ct).ConfigureAwait(false);

                    var writer = new StreamWriter(server, new UTF8Encoding(false)) { AutoFlush = true };
                    lock (_streamLock)
                    {
                        _activeStream = server;
                        _activeWriter = writer;
                    }

                    SetConnectionStatus("CONNECTED // MT5 LIVE", isConnected: true);
                    AppendLog("MT5 EA connected to pipe.", LogLevel.Net);

                    using var reader = new StreamReader(server, new UTF8Encoding(false));
                    while (!ct.IsCancellationRequested && server.IsConnected)
                    {
                        string? line;
                        try
                        {
                            line = await reader.ReadLineAsync(ct).ConfigureAwait(false);
                        }
                        catch (OperationCanceledException) { break; }
                        catch (IOException ioex)
                        {
                            AppendLog("Pipe I/O error: " + ioex.Message, LogLevel.Error);
                            break;
                        }

                        if (line == null) break; // remote closed
                        if (line.Length == 0) continue;

                        ProcessIncomingPacket(line);
                    }
                }
                catch (OperationCanceledException) { /* shutdown */ }
                catch (Exception ex)
                {
                    AppendLog("Pipe loop fault: " + ex.Message, LogLevel.Error);
                }
                finally
                {
                    lock (_streamLock)
                    {
                        try { _activeWriter?.Dispose(); } catch { }
                        try { _activeStream?.Dispose(); } catch { }
                        _activeWriter = null;
                        _activeStream = null;
                    }
                    server?.Dispose();
                    SetConnectionStatus("DISCONNECTED", isConnected: false);
                    if (!ct.IsCancellationRequested)
                        AppendLog("MT5 disconnected. Strict rules preserved. Awaiting reconnect.", LogLevel.Warn);
                }

                if (!ct.IsCancellationRequested)
                {
                    try { await Task.Delay(250, ct).ConfigureAwait(false); }
                    catch (OperationCanceledException) { break; }
                }
            }
        }

        private void SetConnectionStatus(string text, bool isConnected)
        {
            UiInvoke(() =>
            {
                lblConn.Text = text;
                lblConn.ForeColor = isConnected
                    ? System.Drawing.ColorTranslator.FromHtml("#00E6C3")
                    : System.Drawing.ColorTranslator.FromHtml("#F5C451");
            });
        }

        // ============================================================
        // Packet processing
        // ============================================================
        private void ProcessIncomingPacket(string raw)
        {
            if (string.IsNullOrWhiteSpace(raw)) return;
            var line = raw.Trim();

            if (!line.StartsWith("METRICS;", StringComparison.OrdinalIgnoreCase))
            {
                AppendLog("Ignored non-METRICS packet: " + Truncate(line, 80), LogLevel.Warn);
                return;
            }

            if (!AccountMetrics.TryParse(line, out var metrics, out var parseError))
            {
                AppendLog("Parse fail: " + parseError + " // raw=" + Truncate(line, 80), LogLevel.Error);
                return;
            }

            _lastMetrics = metrics;
            EvaluateAndRender(metrics);
        }

        private void EvaluateAndRender(AccountMetrics m)
        {
            // Risk math
            decimal ddPct = 0M;
            if (m.StartingBalance > 0M)
                ddPct = ((m.StartingBalance - m.CurrentEquity) / m.StartingBalance) * 100M;

            decimal floatingPnl = m.CurrentEquity - m.CurrentBalance;

            // Snapshot rules to evaluate consistently
            decimal maxDdPct; decimal maxAbsDd; int maxPos;
            lock (_rulesLock)
            {
                maxDdPct = _maxDailyDdPct;
                maxAbsDd = _maxAbsoluteDd;
                maxPos = _maxOpenPositions;
            }

            bool breachDailyPct = ddPct >= maxDdPct;
            bool breachAbsolute = m.CurrentEquity <= (m.StartingBalance - maxAbsDd);
            bool breachPosCount = m.OpenPositionsCount > maxPos;

            // Render
            UiInvoke(() =>
            {
                lblStartBalance.Text = m.StartingBalance.ToString("N2", CultureInfo.InvariantCulture);
                lblCurBalance.Text = m.CurrentBalance.ToString("N2", CultureInfo.InvariantCulture);
                lblEquity.Text = m.CurrentEquity.ToString("N2", CultureInfo.InvariantCulture);
                lblFloatPnl.Text = (floatingPnl >= 0 ? "+" : "") + floatingPnl.ToString("N2", CultureInfo.InvariantCulture);
                lblFloatPnl.ForeColor = floatingPnl >= 0
                    ? System.Drawing.ColorTranslator.FromHtml("#00E6C3")
                    : System.Drawing.ColorTranslator.FromHtml("#FF8A6B");

                lblDdPct.Text = ddPct.ToString("N2", CultureInfo.InvariantCulture) + " %";
                if (ddPct >= maxDdPct * 0.75M)
                    lblDdPct.ForeColor = System.Drawing.ColorTranslator.FromHtml("#F5C451");
                else
                    lblDdPct.ForeColor = System.Drawing.ColorTranslator.FromHtml("#00E6C3");
                if (ddPct >= maxDdPct)
                    lblDdPct.ForeColor = System.Drawing.ColorTranslator.FromHtml("#FF3B30");
            });

            if (_isLocked) return; // breach already latched

            if (breachDailyPct || breachAbsolute || breachPosCount)
            {
                var reason = breachDailyPct
                    ? $"DAILY_DD {ddPct:N2}% >= {maxDdPct:N2}%"
                    : breachAbsolute
                        ? $"ABS_DD equity {m.CurrentEquity:N2} <= floor {(m.StartingBalance - maxAbsDd):N2}"
                        : $"POS_COUNT {m.OpenPositionsCount} > {maxPos}";
                TriggerBreach(reason);
            }
        }

        // ============================================================
        // Breach execution
        // ============================================================
        private void TriggerBreach(string reason)
        {
            if (_isLocked) return;
            _isLocked = true;

            AppendLog("BREACH: " + reason, LogLevel.Breach);
            AppendLog("Broadcasting KILL_SWITCH to MT5...", LogLevel.Breach);

            bool sent = SendCommand(KillCommand);
            AppendLog(sent
                ? "KILL_SWITCH dispatched successfully."
                : "KILL_SWITCH dispatch FAILED — no active pipe client. Lockout remains engaged.",
                sent ? LogLevel.Breach : LogLevel.Error);

            LockUi();
        }

        private bool SendCommand(string payload)
        {
            lock (_streamLock)
            {
                if (_activeWriter == null || _activeStream == null || !_activeStream.IsConnected)
                    return false;
                try
                {
                    _activeWriter.Write(payload);
                    _activeWriter.Flush();
                    return true;
                }
                catch (Exception ex)
                {
                    AppendLog("SendCommand failed: " + ex.Message, LogLevel.Error);
                    return false;
                }
            }
        }

        private void LockUi()
        {
            UiInvoke(() =>
            {
                var crimson = System.Drawing.ColorTranslator.FromHtml("#FF3B30");
                var deepRed = System.Drawing.ColorTranslator.FromHtml("#2A0E12");

                lblStatus.Text = "● BREACHED / LOCKED";
                lblStatus.ForeColor = crimson;

                // Disable all inputs
                numMaxDdPct.Enabled = false;
                numMaxAbsDd.Enabled = false;
                numMaxLot.Enabled = false;
                numMaxPos.Enabled = false;
                btnApplyRules.Enabled = false;
                btnKillSwitch.Enabled = false;
                btnConnect.Enabled = false;

                // Recolor lock indicator
                pnlLockIndicator.BackColor = crimson;
                lblLockState.ForeColor = System.Drawing.Color.White;
                lblLockState.Text = "● LOCKED  //  KILL SWITCH ENGAGED";

                // Background accent
                BackColor = deepRed;

                // Big flashing warning
                lblBigWarning.Text = "ACCOUNT LOCKED TO PROTECT EQUITY";
                lblBigWarning.Visible = true;

                StartWarningFlasher();
            });
        }

        private System.Windows.Forms.Timer? _flashTimer;
        private void StartWarningFlasher()
        {
            if (_flashTimer != null) return;
            _flashTimer = new System.Windows.Forms.Timer { Interval = 500 };
            bool on = true;
            _flashTimer.Tick += (_, _) =>
            {
                on = !on;
                lblBigWarning.ForeColor = on
                    ? System.Drawing.ColorTranslator.FromHtml("#FF3B30")
                    : System.Drawing.Color.White;
            };
            _flashTimer.Start();
        }

        // ============================================================
        // Utilities
        // ============================================================
        private static string Truncate(string s, int n) => s.Length <= n ? s : s.Substring(0, n) + "...";
    }

    /// <summary>
    /// Strict, allocation-light parser for the MT5 EA wire format:
    ///   METRICS;[AccountID];[StartingBalance];[CurrentBalance];[CurrentEquity];[OpenPositionsCount]
    /// All numeric fields use InvariantCulture (dot decimal separator).
    /// </summary>
    public readonly struct AccountMetrics
    {
        public readonly long AccountId;
        public readonly decimal StartingBalance;
        public readonly decimal CurrentBalance;
        public readonly decimal CurrentEquity;
        public readonly int OpenPositionsCount;

        private AccountMetrics(long id, decimal start, decimal bal, decimal eq, int pos)
        {
            AccountId = id;
            StartingBalance = start;
            CurrentBalance = bal;
            CurrentEquity = eq;
            OpenPositionsCount = pos;
        }

        public static bool TryParse(string line, out AccountMetrics metrics, out string error)
        {
            metrics = default;
            error = string.Empty;
            if (string.IsNullOrWhiteSpace(line))
            {
                error = "empty";
                return false;
            }

            var tokens = line.Split(';');
            if (tokens.Length < 6)
            {
                error = "expected 6 tokens, got " + tokens.Length;
                return false;
            }
            if (!string.Equals(tokens[0], "METRICS", StringComparison.OrdinalIgnoreCase))
            {
                error = "header not METRICS";
                return false;
            }

            if (!long.TryParse(tokens[1], NumberStyles.Integer, CultureInfo.InvariantCulture, out var accId))
            { error = "bad AccountID"; return false; }
            if (!decimal.TryParse(tokens[2], NumberStyles.Float, CultureInfo.InvariantCulture, out var sb))
            { error = "bad StartingBalance"; return false; }
            if (!decimal.TryParse(tokens[3], NumberStyles.Float, CultureInfo.InvariantCulture, out var cb))
            { error = "bad CurrentBalance"; return false; }
            if (!decimal.TryParse(tokens[4], NumberStyles.Float, CultureInfo.InvariantCulture, out var ce))
            { error = "bad CurrentEquity"; return false; }
            if (!int.TryParse(tokens[5], NumberStyles.Integer, CultureInfo.InvariantCulture, out var pos))
            { error = "bad OpenPositionsCount"; return false; }

            metrics = new AccountMetrics(accId, sb, cb, ce, pos);
            return true;
        }
    }
}
