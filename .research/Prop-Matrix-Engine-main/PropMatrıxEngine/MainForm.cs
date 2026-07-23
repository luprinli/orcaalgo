using System.ComponentModel;
using System.Diagnostics;

namespace PropMatrixEngine;

public sealed class MainForm : Form
{
    private readonly AccountRegistry _registry = new();
    private readonly RiskEngine _riskEngine = new(5m, 0.50m);
    private readonly BindingList<AccountRow> _accountRows = new();
    private readonly BindingList<OrderRow> _orderRows = new();
    private readonly CancellationTokenSource _shutdownCts = new();

    private readonly DataGridView _accountsGrid = new();
    private readonly DataGridView _ordersGrid = new();
    private readonly TextBox _eventLog = new();
    private readonly TextBox _mt5AccountIdInput = new();
    private readonly TextBox _mt5PasswordInput = new();
    private readonly ComboBox _serverSelector = new();
    private readonly Label _globalEquityLabel = new();
    private readonly Label _drawdownLabel = new();
    private readonly Label _statusLabel = new();
    private readonly NumericUpDown _drawdownLimitInput = new();
    private readonly NumericUpDown _preBreachBufferInput = new();
    private readonly NumericUpDown _defaultVolumeInput = new();
    private readonly CheckBox _autoKillSwitchInput = new();
    private readonly CheckBox _copyLinesInput = new();
    private readonly Button _killSwitchButton = new();

    private bool _killSwitchInFlight;
    private readonly Color _surfaceColor = Color.FromArgb(17, 20, 29);
    private readonly Color _panelColor = Color.FromArgb(25, 30, 43);
    private readonly Color _accentColor = Color.FromArgb(60, 134, 231);
    private readonly Color _textColor = Color.FromArgb(229, 233, 242);

    public MainForm()
    {
        Text = "Prop Matrix Engine";
        MinimumSize = new Size(1280, 760);
        StartPosition = FormStartPosition.CenterScreen;
        Font = new Font("Segoe UI", 9F);

        BuildUi();
        WireEvents();
    }

    protected override async void OnLoad(EventArgs e)
    {
        base.OnLoad(e);
        await Task.CompletedTask;
        Log("Dashboard armed. Add MT5 credentials to start a formal terminal session.");
    }

    protected override async void OnFormClosing(FormClosingEventArgs e)
    {
        _shutdownCts.Cancel();
        await _registry.DisposeAsync();
        _shutdownCts.Dispose();
        base.OnFormClosing(e);
    }

    private void BuildUi()
    {
        var root = new TableLayoutPanel
        {
            Dock = DockStyle.Fill,
            ColumnCount = 1,
            RowCount = 5,
            Padding = new Padding(14),
            BackColor = _surfaceColor
        };
        root.RowStyles.Add(new RowStyle(SizeType.Absolute, 68));
        root.RowStyles.Add(new RowStyle(SizeType.Absolute, 126));
        root.RowStyles.Add(new RowStyle(SizeType.Percent, 54));
        root.RowStyles.Add(new RowStyle(SizeType.Percent, 25));
        root.RowStyles.Add(new RowStyle(SizeType.Percent, 21));
        Controls.Add(root);

        root.Controls.Add(BuildTitleBar(), 0, 0);
        root.Controls.Add(BuildHeader(), 0, 1);
        root.Controls.Add(BuildAccountsGrid(), 0, 2);
        root.Controls.Add(BuildOrdersGrid(), 0, 3);
        root.Controls.Add(BuildEventLog(), 0, 4);
    }

    private Control BuildTitleBar()
    {
        var panel = new Panel
        {
            Dock = DockStyle.Fill,
            BackColor = Color.FromArgb(12, 15, 22),
            Padding = new Padding(18, 10, 18, 10),
            Margin = new Padding(0, 0, 0, 10)
        };

        var title = new Label
        {
            Text = "PROP MATRIX ENGINE",
            Dock = DockStyle.Left,
            Width = 340,
            ForeColor = Color.White,
            Font = new Font("Segoe UI Semibold", 18F, FontStyle.Bold),
            TextAlign = ContentAlignment.MiddleLeft
        };

        var subtitle = new Label
        {
            Text = "Multi-account equity synchronization and risk command center",
            Dock = DockStyle.Fill,
            ForeColor = Color.FromArgb(151, 164, 185),
            Font = new Font("Segoe UI", 10F),
            TextAlign = ContentAlignment.MiddleLeft
        };

        panel.Controls.Add(subtitle);
        panel.Controls.Add(title);
        return panel;
    }

    private Control BuildHeader()
    {
        var header = new TableLayoutPanel
        {
            Dock = DockStyle.Fill,
            ColumnCount = 3,
            RowCount = 1,
            BackColor = _panelColor,
            Padding = new Padding(14),
            Margin = new Padding(0, 0, 0, 10)
        };
        header.ColumnStyles.Add(new ColumnStyle(SizeType.Percent, 35));
        header.ColumnStyles.Add(new ColumnStyle(SizeType.Percent, 40));
        header.ColumnStyles.Add(new ColumnStyle(SizeType.Percent, 25));

        header.Controls.Add(BuildConnectionPanel(), 0, 0);
        header.Controls.Add(BuildRiskPanel(), 1, 0);
        header.Controls.Add(BuildActionPanel(), 2, 0);

        return header;
    }

    private Control BuildConnectionPanel()
    {
        var panel = new TableLayoutPanel
        {
            Dock = DockStyle.Fill,
            ColumnCount = 4,
            RowCount = 3
        };
        panel.ColumnStyles.Add(new ColumnStyle(SizeType.Absolute, 110));
        panel.ColumnStyles.Add(new ColumnStyle(SizeType.Percent, 50));
        panel.ColumnStyles.Add(new ColumnStyle(SizeType.Absolute, 90));
        panel.ColumnStyles.Add(new ColumnStyle(SizeType.Percent, 50));

        _mt5PasswordInput.UseSystemPasswordChar = true;
        _serverSelector.DropDownStyle = ComboBoxStyle.DropDownList;
        _serverSelector.Items.AddRange(new object[]
        {
            "MetaQuotes-Demo",
            "Broker-London",
            "Broker-NewYork",
            "Broker-Tokyo"
        });
        _serverSelector.SelectedIndex = 0;

        AddLabeledControl(panel, "MT5 ID", _mt5AccountIdInput, 0, 0);
        AddLabeledControl(panel, "Password", _mt5PasswordInput, 2, 0);
        AddLabeledControl(panel, "Server", _serverSelector, 0, 1, 3);

        var connectButton = CreateButton("Add MT5 Account", _accentColor);
        connectButton.Click += async (_, _) => await AddCredentialAccountAsync();

        panel.Controls.Add(connectButton, 1, 2);
        panel.SetColumnSpan(connectButton, 3);
        return panel;
    }

    private Control BuildRiskPanel()
    {
        var panel = new TableLayoutPanel
        {
            Dock = DockStyle.Fill,
            ColumnCount = 4,
            RowCount = 3
        };
        panel.ColumnStyles.Add(new ColumnStyle(SizeType.Absolute, 130));
        panel.ColumnStyles.Add(new ColumnStyle(SizeType.Percent, 40));
        panel.ColumnStyles.Add(new ColumnStyle(SizeType.Absolute, 130));
        panel.ColumnStyles.Add(new ColumnStyle(SizeType.Percent, 60));

        _globalEquityLabel.Text = "Global Equity: 0.00";
        _drawdownLabel.Text = "Daily DD: 0.00%";
        _statusLabel.Text = "Risk Engine: Armed";

        foreach (var label in new[] { _globalEquityLabel, _drawdownLabel, _statusLabel })
        {
            label.Dock = DockStyle.Fill;
            label.ForeColor = _textColor;
            label.TextAlign = ContentAlignment.MiddleLeft;
            label.Font = new Font(Font, FontStyle.Bold);
        }

        _drawdownLimitInput.DecimalPlaces = 2;
        _drawdownLimitInput.Minimum = 0.10m;
        _drawdownLimitInput.Maximum = 100m;
        _drawdownLimitInput.Value = 5.00m;
        _drawdownLimitInput.Increment = 0.25m;

        _preBreachBufferInput.DecimalPlaces = 2;
        _preBreachBufferInput.Minimum = 0m;
        _preBreachBufferInput.Maximum = 10m;
        _preBreachBufferInput.Value = 0.50m;
        _preBreachBufferInput.Increment = 0.10m;

        _defaultVolumeInput.DecimalPlaces = 2;
        _defaultVolumeInput.Minimum = 0.01m;
        _defaultVolumeInput.Maximum = 100m;
        _defaultVolumeInput.Value = 0.10m;
        _defaultVolumeInput.Increment = 0.01m;

        AddLabeledControl(panel, "DD Limit %", _drawdownLimitInput, 0, 0);
        AddLabeledControl(panel, "Pre-Breach %", _preBreachBufferInput, 2, 0);
        AddLabeledControl(panel, "Copy Lots", _defaultVolumeInput, 0, 1);
        panel.Controls.Add(_globalEquityLabel, 2, 1);
        panel.SetColumnSpan(_globalEquityLabel, 2);
        panel.Controls.Add(_drawdownLabel, 0, 2);
        panel.SetColumnSpan(_drawdownLabel, 2);
        panel.Controls.Add(_statusLabel, 2, 2);
        panel.SetColumnSpan(_statusLabel, 2);

        return panel;
    }

    private Control BuildActionPanel()
    {
        var panel = new TableLayoutPanel
        {
            Dock = DockStyle.Fill,
            ColumnCount = 1,
            RowCount = 3
        };
        panel.RowStyles.Add(new RowStyle(SizeType.Percent, 33));
        panel.RowStyles.Add(new RowStyle(SizeType.Percent, 33));
        panel.RowStyles.Add(new RowStyle(SizeType.Percent, 34));

        _autoKillSwitchInput.Text = "Auto kill-switch near daily DD limit";
        _autoKillSwitchInput.Checked = true;
        _copyLinesInput.Text = "Copy manual chart lines as pending orders";
        _copyLinesInput.Checked = true;

        foreach (var checkBox in new[] { _autoKillSwitchInput, _copyLinesInput })
        {
            checkBox.Dock = DockStyle.Fill;
            checkBox.ForeColor = _textColor;
            checkBox.BackColor = Color.Transparent;
        }

        _killSwitchButton.Text = "GLOBAL KILL SWITCH";
        _killSwitchButton.Dock = DockStyle.Fill;
        _killSwitchButton.FlatStyle = FlatStyle.Flat;
        _killSwitchButton.BackColor = Color.FromArgb(169, 39, 39);
        _killSwitchButton.ForeColor = Color.White;
        _killSwitchButton.FlatAppearance.BorderColor = Color.FromArgb(234, 92, 92);
        _killSwitchButton.Font = new Font(Font.FontFamily, 11F, FontStyle.Bold);
        _killSwitchButton.Click += async (_, _) => await ExecuteKillSwitchAsync("Manual operator action");

        panel.Controls.Add(_autoKillSwitchInput, 0, 0);
        panel.Controls.Add(_copyLinesInput, 0, 1);
        panel.Controls.Add(_killSwitchButton, 0, 2);
        return panel;
    }

    private Control BuildAccountsGrid()
    {
        ConfigureGrid(_accountsGrid);
        _accountsGrid.DataSource = _accountRows;
        _accountsGrid.Columns.Add(TextColumn(nameof(AccountRow.DisplayName), "Account", 160));
        _accountsGrid.Columns.Add(TextColumn(nameof(AccountRow.Server), "Server", 140));
        _accountsGrid.Columns.Add(TextColumn(nameof(AccountRow.Login), "Login", 95));
        _accountsGrid.Columns.Add(TextColumn(nameof(AccountRow.Mode), "Mode", 90));
        _accountsGrid.Columns.Add(TextColumn(nameof(AccountRow.Status), "Status", 95));
        _accountsGrid.Columns.Add(TextColumn(nameof(AccountRow.Balance), "Balance", 110));
        _accountsGrid.Columns.Add(TextColumn(nameof(AccountRow.Equity), "Equity", 110));
        _accountsGrid.Columns.Add(TextColumn(nameof(AccountRow.FloatingProfitLoss), "Floating P/L", 110));
        _accountsGrid.Columns.Add(TextColumn(nameof(AccountRow.Exposure), "Exposure", 110));
        _accountsGrid.Columns.Add(TextColumn(nameof(AccountRow.LastTick), "Last Tick", 160));
        return _accountsGrid;
    }

    private Control BuildOrdersGrid()
    {
        ConfigureGrid(_ordersGrid);
        _ordersGrid.DataSource = _orderRows;
        _ordersGrid.Columns.Add(TextColumn(nameof(OrderRow.Time), "Time", 150));
        _ordersGrid.Columns.Add(TextColumn(nameof(OrderRow.AccountId), "Account", 120));
        _ordersGrid.Columns.Add(TextColumn(nameof(OrderRow.Type), "Command", 140));
        _ordersGrid.Columns.Add(TextColumn(nameof(OrderRow.Success), "OK", 50));
        _ordersGrid.Columns.Add(TextColumn(nameof(OrderRow.ElapsedMs), "ms", 70));
        _ordersGrid.Columns.Add(TextColumn(nameof(OrderRow.Message), "Message", 620));
        return _ordersGrid;
    }

    private Control BuildEventLog()
    {
        _eventLog.Dock = DockStyle.Fill;
        _eventLog.Multiline = true;
        _eventLog.ScrollBars = ScrollBars.Vertical;
        _eventLog.ReadOnly = true;
        _eventLog.BackColor = Color.FromArgb(9, 12, 18);
        _eventLog.ForeColor = Color.FromArgb(197, 207, 224);
        _eventLog.BorderStyle = BorderStyle.FixedSingle;
        return _eventLog;
    }

    private void WireEvents()
    {
        _registry.SnapshotReceived += (_, snapshot) => Post(() =>
        {
            UpsertAccount(snapshot);
            RefreshRisk();
        });

        _registry.ChartLineReceived += (_, signal) => Post(async () => await HandleChartLineAsync(signal));
        _registry.CommandCompleted += (_, result) => Post(() => AddCommandResult(result));
        _riskEngine.KillSwitchRequired += (_, risk) => Post(async () =>
        {
            _statusLabel.Text = $"Risk Engine: TRIGGERED - {risk.Reason}";
            _statusLabel.ForeColor = Color.FromArgb(255, 188, 88);

            if (_autoKillSwitchInput.Checked)
            {
                await ExecuteKillSwitchAsync(risk.Reason);
            }
        });

        _drawdownLimitInput.ValueChanged += (_, _) => RefreshRisk();
        _preBreachBufferInput.ValueChanged += (_, _) => RefreshRisk();
    }

    private async Task AddCredentialAccountAsync()
    {
        var mt5AccountId = _mt5AccountIdInput.Text.Trim();
        var password = _mt5PasswordInput.Text;

        if (string.IsNullOrWhiteSpace(mt5AccountId) || string.IsNullOrWhiteSpace(password))
        {
            MessageBox.Show(this, "MT5 ID and password are required.", Text, MessageBoxButtons.OK, MessageBoxIcon.Warning);
            return;
        }

        if (_registry.Clients.Any(client => client.Endpoint.AccountId == mt5AccountId))
        {
            Log($"MT5 account {mt5AccountId} is already active.");
            _mt5PasswordInput.Clear();
            return;
        }

        var endpoint = new AccountEndpoint(
            mt5AccountId,
            $"MT5 {mt5AccountId}",
            _serverSelector.SelectedItem?.ToString() ?? "MetaQuotes-Demo",
            "mt5://credential-session",
            Mt5ConnectionMode.Simulation);

        await RegisterEndpointAsync(endpoint);
        _mt5PasswordInput.Clear();
    }

    private async Task RegisterEndpointAsync(AccountEndpoint endpoint)
    {
        try
        {
            Log($"Connecting {endpoint.DisplayName} via {endpoint.Mode}...");
            await _registry.AddAndConnectAsync(endpoint, _shutdownCts.Token);
            Log($"{endpoint.DisplayName} connected.");
        }
        catch (Exception ex)
        {
            Log($"Connection failed for {endpoint.DisplayName}: {ex.Message}");
            MessageBox.Show(this, ex.Message, "Connection failed", MessageBoxButtons.OK, MessageBoxIcon.Error);
        }
    }

    private async Task HandleChartLineAsync(ChartLineSignal signal)
    {
        Log($"Chart line detected on {signal.SourceAccountId}: {signal.Symbol} {signal.Price:N5} '{signal.Label}'.");

        if (!_copyLinesInput.Checked)
        {
            return;
        }

        try
        {
            var results = await _registry.CopyChartLineAsPendingOrdersAsync(
                signal,
                _defaultVolumeInput.Value,
                _shutdownCts.Token);

            Log($"Replicated chart line to {results.Count} account(s) as asymmetric pending orders.");
        }
        catch (Exception ex)
        {
            Log($"Chart-line copy failed: {ex.Message}");
        }
    }

    private async Task ExecuteKillSwitchAsync(string reason)
    {
        if (_killSwitchInFlight)
        {
            return;
        }

        _killSwitchInFlight = true;
        _killSwitchButton.Enabled = false;
        _killSwitchButton.Text = "KILL SWITCH FIRING...";
        Log($"Kill-switch firing: {reason}");

        var stopwatch = Stopwatch.StartNew();
        try
        {
            var results = await _registry.ExecuteGlobalKillSwitchAsync(_shutdownCts.Token);
            stopwatch.Stop();
            Log($"Kill-switch completed: {results.Count} command acknowledgements in {stopwatch.ElapsedMilliseconds} ms.");
        }
        catch (Exception ex)
        {
            Log($"Kill-switch failed: {ex.Message}");
        }
        finally
        {
            _killSwitchInFlight = false;
            _killSwitchButton.Enabled = true;
            _killSwitchButton.Text = "GLOBAL KILL SWITCH";
        }
    }

    private void RefreshRisk()
    {
        _riskEngine.DrawdownLimitPercent = _drawdownLimitInput.Value;
        _riskEngine.PreBreachBufferPercent = _preBreachBufferInput.Value;

        var risk = _riskEngine.Evaluate(_registry.Snapshots);
        _globalEquityLabel.Text = $"Global Equity: {risk.TotalEquity:N2}";
        _drawdownLabel.Text = $"Daily DD: {risk.DrawdownPercent:N2}% / {risk.DrawdownLimitPercent:N2}%";

        if (!risk.IsKillSwitchRequired)
        {
            _statusLabel.Text = risk.IsArmed ? "Risk Engine: Armed" : "Risk Engine: Disarmed";
            _statusLabel.ForeColor = _textColor;
        }
    }

    private void UpsertAccount(AccountSnapshot snapshot)
    {
        var existing = _accountRows.FirstOrDefault(row => row.AccountId == snapshot.AccountId);
        if (existing is null)
        {
            _accountRows.Add(AccountRow.From(snapshot, ResolveMode(snapshot.AccountId)));
        }
        else
        {
            existing.Apply(snapshot, ResolveMode(snapshot.AccountId));
            _accountsGrid.Refresh();
        }
    }

    private string ResolveMode(string accountId) =>
        _registry.Clients.FirstOrDefault(client => client.Endpoint.AccountId == accountId)?.Endpoint.Mode.ToString()
        ?? "Unknown";

    private void AddCommandResult(TradeCommandResult result)
    {
        _orderRows.Insert(0, OrderRow.From(result));
        while (_orderRows.Count > 250)
        {
            _orderRows.RemoveAt(_orderRows.Count - 1);
        }
    }

    private void Log(string message)
    {
        _eventLog.AppendText($"[{DateTime.Now:HH:mm:ss.fff}] {message}{Environment.NewLine}");
    }

    private void Post(Action action)
    {
        if (IsDisposed || Disposing)
        {
            return;
        }

        if (InvokeRequired)
        {
            BeginInvoke(action);
            return;
        }

        action();
    }

    private static void ConfigureGrid(DataGridView grid)
    {
        grid.Dock = DockStyle.Fill;
        grid.AutoGenerateColumns = false;
        grid.AllowUserToAddRows = false;
        grid.AllowUserToDeleteRows = false;
        grid.ReadOnly = true;
        grid.RowHeadersVisible = false;
        grid.SelectionMode = DataGridViewSelectionMode.FullRowSelect;
        grid.BackgroundColor = Color.FromArgb(9, 12, 18);
        grid.BorderStyle = BorderStyle.FixedSingle;
        grid.EnableHeadersVisualStyles = false;
        grid.ColumnHeadersDefaultCellStyle.BackColor = Color.FromArgb(30, 37, 54);
        grid.ColumnHeadersDefaultCellStyle.ForeColor = Color.White;
        grid.ColumnHeadersDefaultCellStyle.Font = new Font("Segoe UI Semibold", 9F, FontStyle.Bold);
        grid.DefaultCellStyle.BackColor = Color.FromArgb(14, 18, 27);
        grid.DefaultCellStyle.ForeColor = Color.FromArgb(229, 233, 242);
        grid.DefaultCellStyle.SelectionBackColor = Color.FromArgb(60, 134, 231);
        grid.DefaultCellStyle.SelectionForeColor = Color.White;
        grid.AlternatingRowsDefaultCellStyle.BackColor = Color.FromArgb(18, 23, 34);
        grid.GridColor = Color.FromArgb(43, 51, 68);
        grid.AutoSizeColumnsMode = DataGridViewAutoSizeColumnsMode.Fill;
    }

    private static DataGridViewTextBoxColumn TextColumn(string propertyName, string header, int width) =>
        new()
        {
            DataPropertyName = propertyName,
            HeaderText = header,
            MinimumWidth = Math.Min(width, 90),
            Width = width
        };

    private static Button CreateButton(string text, Color backColor) =>
        new()
        {
            Text = text,
            Dock = DockStyle.Fill,
            FlatStyle = FlatStyle.Flat,
            BackColor = backColor,
            ForeColor = Color.White,
            Margin = new Padding(4),
            Font = new Font("Segoe UI Semibold", 9F, FontStyle.Bold)
        };

    private static void AddLabeledControl(
        TableLayoutPanel panel,
        string labelText,
        Control control,
        int column,
        int row,
        int controlColumnSpan = 1)
    {
        var label = new Label
        {
            Text = labelText,
            Dock = DockStyle.Fill,
            ForeColor = Color.FromArgb(188, 196, 208),
            TextAlign = ContentAlignment.MiddleLeft
        };

        control.Dock = DockStyle.Fill;
        control.Margin = new Padding(4);
        ApplyInputStyle(control);

        panel.Controls.Add(label, column, row);
        panel.Controls.Add(control, column + 1, row);
        if (controlColumnSpan > 1)
        {
            panel.SetColumnSpan(control, controlColumnSpan);
        }
    }

    private static string EmptyToDefault(string value, string fallback) =>
        string.IsNullOrWhiteSpace(value) ? fallback : value.Trim();

    private static void ApplyInputStyle(Control control)
    {
        control.BackColor = Color.FromArgb(10, 13, 20);
        control.ForeColor = Color.FromArgb(229, 233, 242);
        control.Font = new Font("Segoe UI", 9F);

        if (control is TextBox textBox)
        {
            textBox.BorderStyle = BorderStyle.FixedSingle;
        }
        else if (control is NumericUpDown numericUpDown)
        {
            numericUpDown.BorderStyle = BorderStyle.FixedSingle;
        }
    }

    private sealed class AccountRow
    {
        public string AccountId { get; private set; } = "";
        public string DisplayName { get; private set; } = "";
        public string Server { get; private set; } = "";
        public string Login { get; private set; } = "";
        public string Mode { get; private set; } = "";
        public string Status { get; private set; } = "";
        public string Balance { get; private set; } = "";
        public string Equity { get; private set; } = "";
        public string FloatingProfitLoss { get; private set; } = "";
        public string Exposure { get; private set; } = "";
        public string LastTick { get; private set; } = "";

        public static AccountRow From(AccountSnapshot snapshot, string mode)
        {
            var row = new AccountRow();
            row.Apply(snapshot, mode);
            return row;
        }

        public void Apply(AccountSnapshot snapshot, string mode)
        {
            AccountId = snapshot.AccountId;
            DisplayName = snapshot.DisplayName;
            Server = snapshot.Server;
            Login = snapshot.Login.ToString();
            Mode = mode;
            Status = snapshot.Status.ToString();
            Balance = snapshot.Balance.ToString("N2");
            Equity = snapshot.Equity.ToString("N2");
            FloatingProfitLoss = snapshot.FloatingProfitLoss.ToString("N2");
            Exposure = snapshot.Exposure.ToString("N2");
            LastTick = snapshot.ServerTime.LocalDateTime.ToString("yyyy-MM-dd HH:mm:ss.fff");
        }
    }

    private sealed class OrderRow
    {
        public string Time { get; init; } = "";
        public string AccountId { get; init; } = "";
        public string Type { get; init; } = "";
        public string Success { get; init; } = "";
        public string ElapsedMs { get; init; } = "";
        public string Message { get; init; } = "";

        public static OrderRow From(TradeCommandResult result) =>
            new()
            {
                Time = DateTime.Now.ToString("HH:mm:ss.fff"),
                AccountId = result.AccountId,
                Type = result.Type.ToString(),
                Success = result.Success ? "Yes" : "No",
                ElapsedMs = result.Elapsed.TotalMilliseconds.ToString("N0"),
                Message = result.Message
            };
    }
}
