namespace PropFirmDrawdown
{
    partial class Form1
    {
        private System.ComponentModel.IContainer components = null;

        // Sidebar
        private Panel pnlSidebar;
        private Label lblLogo;
        private Label lblLogoSub;
        private Label lblStatusTitle;
        private Label lblStatus;
        private Label lblConnTitle;
        private Label lblConn;
        private Button btnConnect;

        // Header
        private Panel pnlHeader;
        private Label lblHeaderTitle;
        private Label lblHeaderSub;

        // Card 1 - Live Metrics
        private Panel cardMetrics;
        private Label lblCardMetricsTitle;
        private Label lblStartBalanceCap, lblStartBalance;
        private Label lblCurBalanceCap, lblCurBalance;
        private Label lblEquityCap, lblEquity;
        private Label lblFloatPnlCap, lblFloatPnl;
        private Label lblDdPctCap, lblDdPct;

        // Card 2 - Strict Rules
        private Panel cardRules;
        private Label lblCardRulesTitle;
        private Label lblMaxDdPctCap;
        private NumericUpDown numMaxDdPct;
        private Label lblMaxAbsDdCap;
        private NumericUpDown numMaxAbsDd;
        private Label lblMaxLotCap;
        private NumericUpDown numMaxLot;
        private Label lblMaxPosCap;
        private NumericUpDown numMaxPos;
        private Button btnApplyRules;

        // Card 3 - Breach Action
        private Panel cardAction;
        private Label lblCardActionTitle;
        private Panel pnlLockIndicator;
        private Label lblLockState;
        private Button btnKillSwitch;
        private Label lblBigWarning;

        // Log
        private Panel pnlLog;
        private Label lblLogTitle;
        private RichTextBox rtbLog;

        protected override void Dispose(bool disposing)
        {
            if (disposing && (components != null)) components.Dispose();
            base.Dispose(disposing);
        }

        private void InitializeComponent()
        {
            this.components = new System.ComponentModel.Container();

            var clrBg = System.Drawing.ColorTranslator.FromHtml("#121622");
            var clrCard = System.Drawing.ColorTranslator.FromHtml("#1A1F2C");
            var clrSide = System.Drawing.ColorTranslator.FromHtml("#0F1320");
            var clrText = System.Drawing.ColorTranslator.FromHtml("#E6EAF2");
            var clrMuted = System.Drawing.ColorTranslator.FromHtml("#6B7488");
            var clrTeal = System.Drawing.ColorTranslator.FromHtml("#00E6C3");
            var clrCrimson = System.Drawing.ColorTranslator.FromHtml("#FF3B30");
            var clrAmber = System.Drawing.ColorTranslator.FromHtml("#F5C451");

            var fontLogo = new System.Drawing.Font("Segoe UI", 11F, System.Drawing.FontStyle.Bold);
            var fontLogoSub = new System.Drawing.Font("Segoe UI", 7.5F);
            var fontHeader = new System.Drawing.Font("Segoe UI Semibold", 14F, System.Drawing.FontStyle.Bold);
            var fontHeaderSub = new System.Drawing.Font("Segoe UI", 8F);
            var fontCardTitle = new System.Drawing.Font("Segoe UI Semibold", 10F, System.Drawing.FontStyle.Bold);
            var fontCap = new System.Drawing.Font("Segoe UI", 8.25F);
            var fontVal = new System.Drawing.Font("Segoe UI Semibold", 11F, System.Drawing.FontStyle.Bold);
            var fontTiny = new System.Drawing.Font("Segoe UI", 7.5F);
            var fontWarning = new System.Drawing.Font("Segoe UI Black", 16F, System.Drawing.FontStyle.Bold);

            // ===== Sidebar =====
            this.pnlSidebar = new Panel
            {
                Dock = DockStyle.Left,
                Width = 220,
                BackColor = clrSide
            };

            this.lblLogo = new Label
            {
                Text = "AUTOSCRIPTS GUARD",
                Font = fontLogo,
                ForeColor = clrText,
                AutoSize = false,
                Location = new System.Drawing.Point(18, 18),
                Size = new System.Drawing.Size(190, 22)
            };
            this.lblLogoSub = new Label
            {
                Text = "MT5 // PROP-FIRM DRAWDOWN GUARD",
                Font = fontLogoSub,
                ForeColor = clrMuted,
                AutoSize = false,
                Location = new System.Drawing.Point(18, 40),
                Size = new System.Drawing.Size(190, 14)
            };

            this.lblStatusTitle = new Label
            {
                Text = "GUARD STATUS",
                Font = fontTiny,
                ForeColor = clrMuted,
                AutoSize = false,
                Location = new System.Drawing.Point(18, 90),
                Size = new System.Drawing.Size(180, 14)
            };
            this.lblStatus = new Label
            {
                Text = "● GUARD ACTIVE",
                Font = new System.Drawing.Font("Segoe UI Semibold", 11F, System.Drawing.FontStyle.Bold),
                ForeColor = clrTeal,
                AutoSize = false,
                Location = new System.Drawing.Point(18, 108),
                Size = new System.Drawing.Size(190, 24)
            };

            this.lblConnTitle = new Label
            {
                Text = "PIPE LINK",
                Font = fontTiny,
                ForeColor = clrMuted,
                AutoSize = false,
                Location = new System.Drawing.Point(18, 150),
                Size = new System.Drawing.Size(180, 14)
            };
            this.lblConn = new Label
            {
                Text = "DISCONNECTED",
                Font = new System.Drawing.Font("Segoe UI Semibold", 9.5F, System.Drawing.FontStyle.Bold),
                ForeColor = clrAmber,
                AutoSize = false,
                Location = new System.Drawing.Point(18, 168),
                Size = new System.Drawing.Size(190, 20)
            };

            this.btnConnect = new Button
            {
                Text = "CONNECT TO MT5",
                Font = new System.Drawing.Font("Segoe UI Semibold", 9.5F, System.Drawing.FontStyle.Bold),
                BackColor = clrTeal,
                ForeColor = System.Drawing.Color.Black,
                FlatStyle = FlatStyle.Flat,
                Location = new System.Drawing.Point(18, 210),
                Size = new System.Drawing.Size(184, 38),
                Cursor = Cursors.Hand,
                TextAlign = System.Drawing.ContentAlignment.MiddleCenter
            };
            this.btnConnect.FlatAppearance.BorderSize = 0;
            this.btnConnect.Click += new System.EventHandler(this.BtnConnect_Click);

            this.pnlSidebar.Controls.AddRange(new Control[]
            {
                this.lblLogo, this.lblLogoSub,
                this.lblStatusTitle, this.lblStatus,
                this.lblConnTitle, this.lblConn,
                this.btnConnect
            });

            // ===== Header =====
            this.pnlHeader = new Panel
            {
                Dock = DockStyle.Top,
                Height = 70,
                BackColor = clrBg
            };
            this.lblHeaderTitle = new Label
            {
                Text = "Advanced Prop-Firm Drawdown & Risk Guard",
                Font = fontHeader,
                ForeColor = clrText,
                AutoSize = false,
                Location = new System.Drawing.Point(20, 14),
                Size = new System.Drawing.Size(800, 28)
            };
            this.lblHeaderSub = new Label
            {
                Text = "Strict equity supervisor // Async Named Pipe: \\\\.\\pipe\\AUTOSCRIPTS_RISK_GUARD",
                Font = fontHeaderSub,
                ForeColor = clrMuted,
                AutoSize = false,
                Location = new System.Drawing.Point(20, 42),
                Size = new System.Drawing.Size(800, 18)
            };
            this.pnlHeader.Controls.AddRange(new Control[] { this.lblHeaderTitle, this.lblHeaderSub });

            // ===== Card 1 - Live Metrics =====
            this.cardMetrics = new Panel
            {
                BackColor = clrCard,
                Location = new System.Drawing.Point(20, 90),
                Size = new System.Drawing.Size(330, 280)
            };
            this.lblCardMetricsTitle = new Label
            {
                Text = "● LIVE METRICS",
                Font = fontCardTitle,
                ForeColor = clrTeal,
                AutoSize = false,
                Location = new System.Drawing.Point(14, 12),
                Size = new System.Drawing.Size(300, 22)
            };

            int yMet = 56; int rowH = 42;
            this.lblStartBalanceCap = MakeCap("Starting Balance", 14, yMet, clrMuted, fontCap);
            this.lblStartBalance = MakeVal("0.00", 14, yMet + 14, clrText, fontVal);
            this.lblCurBalanceCap = MakeCap("Current Balance", 14, yMet + rowH, clrMuted, fontCap);
            this.lblCurBalance = MakeVal("0.00", 14, yMet + rowH + 14, clrText, fontVal);
            this.lblEquityCap = MakeCap("Current Equity", 14, yMet + rowH * 2, clrMuted, fontCap);
            this.lblEquity = MakeVal("0.00", 14, yMet + rowH * 2 + 14, clrText, fontVal);
            this.lblFloatPnlCap = MakeCap("Floating PnL", 14, yMet + rowH * 3, clrMuted, fontCap);
            this.lblFloatPnl = MakeVal("0.00", 14, yMet + rowH * 3 + 14, clrText, fontVal);
            this.lblDdPctCap = MakeCap("Daily Floating Drawdown (%)", 14, yMet + rowH * 4, clrMuted, fontCap);
            this.lblDdPct = MakeVal("0.00 %", 14, yMet + rowH * 4 + 14, clrTeal, fontVal);

            this.cardMetrics.Controls.AddRange(new Control[]
            {
                this.lblCardMetricsTitle,
                this.lblStartBalanceCap, this.lblStartBalance,
                this.lblCurBalanceCap, this.lblCurBalance,
                this.lblEquityCap, this.lblEquity,
                this.lblFloatPnlCap, this.lblFloatPnl,
                this.lblDdPctCap, this.lblDdPct
            });

            // ===== Card 2 - Strict Rules =====
            this.cardRules = new Panel
            {
                BackColor = clrCard,
                Location = new System.Drawing.Point(360, 90),
                Size = new System.Drawing.Size(360, 280)
            };
            this.lblCardRulesTitle = new Label
            {
                Text = "● STRICT RULES INPUT",
                Font = fontCardTitle,
                ForeColor = clrTeal,
                AutoSize = false,
                Location = new System.Drawing.Point(14, 12),
                Size = new System.Drawing.Size(300, 22)
            };

            this.lblMaxDdPctCap = MakeCap("Max Daily Drawdown (%)", 14, 50, clrMuted, fontCap);
            this.numMaxDdPct = MakeNumeric(14, 68, 5M, 0.01M, 100M, 2);

            this.lblMaxAbsDdCap = MakeCap("Max Absolute Drawdown ($)", 188, 50, clrMuted, fontCap);
            this.numMaxAbsDd = MakeNumeric(188, 68, 1000M, 1M, 10000000M, 2);

            this.lblMaxLotCap = MakeCap("Max Allowed Lot Size per Trade", 14, 110, clrMuted, fontCap);
            this.numMaxLot = MakeNumeric(14, 128, 1.00M, 0.01M, 1000M, 2);

            this.lblMaxPosCap = MakeCap("Max Allowed Open Positions", 188, 110, clrMuted, fontCap);
            this.numMaxPos = MakeNumeric(188, 128, 5M, 1M, 9999M, 0);

            this.btnApplyRules = new Button
            {
                Text = "APPLY STRICT RULES",
                Font = new System.Drawing.Font("Segoe UI Semibold", 9.5F, System.Drawing.FontStyle.Bold),
                BackColor = clrTeal,
                ForeColor = System.Drawing.Color.Black,
                FlatStyle = FlatStyle.Flat,
                Location = new System.Drawing.Point(14, 220),
                Size = new System.Drawing.Size(332, 38),
                Cursor = Cursors.Hand
            };
            this.btnApplyRules.FlatAppearance.BorderSize = 0;
            this.btnApplyRules.Click += new System.EventHandler(this.BtnApplyRules_Click);

            this.cardRules.Controls.AddRange(new Control[]
            {
                this.lblCardRulesTitle,
                this.lblMaxDdPctCap, this.numMaxDdPct,
                this.lblMaxAbsDdCap, this.numMaxAbsDd,
                this.lblMaxLotCap, this.numMaxLot,
                this.lblMaxPosCap, this.numMaxPos,
                this.btnApplyRules
            });

            // ===== Card 3 - Breach Action =====
            this.cardAction = new Panel
            {
                BackColor = clrCard,
                Location = new System.Drawing.Point(730, 90),
                Size = new System.Drawing.Size(360, 280)
            };
            this.lblCardActionTitle = new Label
            {
                Text = "● BREACH ACTION CONSOLE",
                Font = fontCardTitle,
                ForeColor = clrCrimson,
                AutoSize = false,
                Location = new System.Drawing.Point(14, 12),
                Size = new System.Drawing.Size(330, 22)
            };

            this.pnlLockIndicator = new Panel
            {
                BackColor = System.Drawing.ColorTranslator.FromHtml("#10202A"),
                Location = new System.Drawing.Point(14, 50),
                Size = new System.Drawing.Size(332, 80)
            };
            this.lblLockState = new Label
            {
                Text = "● UNLOCKED  //  EQUITY SAFE",
                Font = new System.Drawing.Font("Segoe UI Black", 12F, System.Drawing.FontStyle.Bold),
                ForeColor = clrTeal,
                AutoSize = false,
                TextAlign = System.Drawing.ContentAlignment.MiddleCenter,
                Dock = DockStyle.Fill
            };
            this.pnlLockIndicator.Controls.Add(this.lblLockState);

            this.btnKillSwitch = new Button
            {
                Text = "FORCE KILL SWITCH  (Emergency Close)",
                Font = new System.Drawing.Font("Segoe UI Black", 10F, System.Drawing.FontStyle.Bold),
                BackColor = clrCrimson,
                ForeColor = System.Drawing.Color.White,
                FlatStyle = FlatStyle.Flat,
                Location = new System.Drawing.Point(14, 150),
                Size = new System.Drawing.Size(332, 56),
                Cursor = Cursors.Hand
            };
            this.btnKillSwitch.FlatAppearance.BorderSize = 0;
            this.btnKillSwitch.Click += new System.EventHandler(this.BtnKillSwitch_Click);

            this.lblBigWarning = new Label
            {
                Text = "",
                Font = fontWarning,
                ForeColor = clrCrimson,
                AutoSize = false,
                TextAlign = System.Drawing.ContentAlignment.MiddleCenter,
                Location = new System.Drawing.Point(14, 220),
                Size = new System.Drawing.Size(332, 46),
                Visible = false
            };

            this.cardAction.Controls.AddRange(new Control[]
            {
                this.lblCardActionTitle,
                this.pnlLockIndicator,
                this.btnKillSwitch,
                this.lblBigWarning
            });

            // ===== Log =====
            this.pnlLog = new Panel
            {
                BackColor = clrCard,
                Location = new System.Drawing.Point(20, 390),
                Size = new System.Drawing.Size(1070, 220)
            };
            this.lblLogTitle = new Label
            {
                Text = "● RISK DIAGNOSTIC LOG",
                Font = fontCardTitle,
                ForeColor = clrTeal,
                AutoSize = false,
                Location = new System.Drawing.Point(14, 10),
                Size = new System.Drawing.Size(300, 22)
            };
            this.rtbLog = new RichTextBox
            {
                BackColor = System.Drawing.ColorTranslator.FromHtml("#0D111B"),
                ForeColor = clrText,
                Font = new System.Drawing.Font("Consolas", 9F),
                BorderStyle = BorderStyle.None,
                ReadOnly = true,
                Location = new System.Drawing.Point(14, 38),
                Size = new System.Drawing.Size(1042, 170),
                DetectUrls = false
            };
            this.pnlLog.Controls.AddRange(new Control[] { this.lblLogTitle, this.rtbLog });

            // ===== Form =====
            this.SuspendLayout();
            this.AutoScaleMode = AutoScaleMode.None;
            this.ClientSize = new System.Drawing.Size(1330, 630);
            this.BackColor = clrBg;
            this.ForeColor = clrText;
            this.Text = "Advanced Prop-Firm Drawdown & Risk Guard";
            this.StartPosition = FormStartPosition.CenterScreen;
            this.FormBorderStyle = FormBorderStyle.FixedSingle;
            this.MaximizeBox = false;
            this.Font = new System.Drawing.Font("Segoe UI", 9F);

            var pnlMain = new Panel { Dock = DockStyle.Fill, BackColor = clrBg };
            pnlMain.Controls.Add(this.cardMetrics);
            pnlMain.Controls.Add(this.cardRules);
            pnlMain.Controls.Add(this.cardAction);
            pnlMain.Controls.Add(this.pnlLog);
            pnlMain.Controls.Add(this.pnlHeader);

            this.Controls.Add(pnlMain);
            this.Controls.Add(this.pnlSidebar);

            this.FormClosing += new FormClosingEventHandler(this.Form1_FormClosing);
            this.Load += new System.EventHandler(this.Form1_Load);

            this.ResumeLayout(false);
        }

        private static Label MakeCap(string text, int x, int y, System.Drawing.Color clr, System.Drawing.Font fnt)
        {
            return new Label
            {
                Text = text,
                Font = fnt,
                ForeColor = clr,
                AutoSize = false,
                Location = new System.Drawing.Point(x, y),
                Size = new System.Drawing.Size(300, 14)
            };
        }

        private static Label MakeVal(string text, int x, int y, System.Drawing.Color clr, System.Drawing.Font fnt)
        {
            return new Label
            {
                Text = text,
                Font = fnt,
                ForeColor = clr,
                AutoSize = false,
                Location = new System.Drawing.Point(x, y),
                Size = new System.Drawing.Size(300, 22)
            };
        }

        private static NumericUpDown MakeNumeric(int x, int y, decimal val, decimal min, decimal max, int dec)
        {
            return new NumericUpDown
            {
                Location = new System.Drawing.Point(x, y),
                Size = new System.Drawing.Size(160, 28),
                Minimum = min,
                Maximum = max,
                DecimalPlaces = dec,
                Increment = dec == 0 ? 1M : 0.1M,
                Value = val,
                BackColor = System.Drawing.ColorTranslator.FromHtml("#0D111B"),
                ForeColor = System.Drawing.ColorTranslator.FromHtml("#E6EAF2"),
                BorderStyle = BorderStyle.FixedSingle,
                Font = new System.Drawing.Font("Segoe UI Semibold", 10F, System.Drawing.FontStyle.Bold)
            };
        }
    }
}
