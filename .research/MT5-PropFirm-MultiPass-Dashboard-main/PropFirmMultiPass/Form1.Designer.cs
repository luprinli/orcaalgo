namespace PropFirmMultiPass
{
    partial class Form1
    {
        private System.ComponentModel.IContainer components = null;

        protected override void Dispose(bool disposing)
        {
            if (disposing && (components != null))
            {
                components.Dispose();
            }
            base.Dispose(disposing);
        }

        #region Windows Form Designer generated code

        private void InitializeComponent()
        {
            components = new System.ComponentModel.Container();

            // === Color Palette ===
            var bgDeep = Color.FromArgb(11, 13, 17);       // #0B0D11
            var bgPanel = Color.FromArgb(18, 23, 30);      // #12171E
            var emerald = Color.FromArgb(0, 230, 118);     // #00E676
            var amber = Color.FromArgb(255, 214, 0);       // #FFD600
            var softRed = Color.FromArgb(207, 102, 121);   // Soft Charcoal-Red
            var dimText = Color.FromArgb(130, 140, 160);
            var brightText = Color.FromArgb(210, 220, 235);

            // === Main Form ===
            SuspendLayout();
            AutoScaleMode = AutoScaleMode.Font;
            ClientSize = new Size(1440, 900);
            Text = "AUTOSCRIPTS STEALTH GUARD \u2014 Prop-Firm Multi-Pass & Hidden Virtual SL/TP Manager";
            BackColor = bgDeep;
            ForeColor = brightText;
            Font = new Font("Segoe UI", 9.5F, FontStyle.Regular, GraphicsUnit.Point);
            StartPosition = FormStartPosition.CenterScreen;
            MinimumSize = new Size(1200, 750);
            DoubleBuffered = true;

            // ===================================================================
            // LEFT SIDEBAR
            // ===================================================================
            pnlSidebar = new Panel();
            pnlSidebar.Dock = DockStyle.Left;
            pnlSidebar.Width = 260;
            pnlSidebar.BackColor = bgPanel;
            pnlSidebar.Padding = new Padding(16, 20, 16, 20);

            lblLogo = new Label();
            lblLogo.Text = "AUTOSCRIPTS\nSTEALTH";
            lblLogo.Font = new Font("Segoe UI", 18F, FontStyle.Bold);
            lblLogo.ForeColor = emerald;
            lblLogo.Dock = DockStyle.Top;
            lblLogo.Height = 80;
            lblLogo.TextAlign = ContentAlignment.MiddleCenter;

            lblSeparator1 = new Label();
            lblSeparator1.Dock = DockStyle.Top;
            lblSeparator1.Height = 1;
            lblSeparator1.BackColor = Color.FromArgb(40, 50, 65);

            pnlStatusCard = new Panel();
            pnlStatusCard.Dock = DockStyle.Top;
            pnlStatusCard.Height = 220;
            pnlStatusCard.BackColor = bgPanel;
            pnlStatusCard.Padding = new Padding(8, 16, 8, 8);

            lblShieldLabel = new Label();
            lblShieldLabel.Text = "STEALTH SHIELD";
            lblShieldLabel.Font = new Font("Segoe UI", 9F, FontStyle.Bold);
            lblShieldLabel.ForeColor = dimText;
            lblShieldLabel.Location = new Point(12, 10);
            lblShieldLabel.AutoSize = true;

            lblShieldStatus = new Label();
            lblShieldStatus.Text = "INACTIVE";
            lblShieldStatus.Font = new Font("Segoe UI", 14F, FontStyle.Bold);
            lblShieldStatus.ForeColor = softRed;
            lblShieldStatus.Location = new Point(12, 32);
            lblShieldStatus.AutoSize = true;

            lblActiveLabel = new Label();
            lblActiveLabel.Text = "ACTIVE STEALTH POSITIONS";
            lblActiveLabel.Font = new Font("Segoe UI", 9F, FontStyle.Bold);
            lblActiveLabel.ForeColor = dimText;
            lblActiveLabel.Location = new Point(12, 78);
            lblActiveLabel.AutoSize = true;

            lblActiveCount = new Label();
            lblActiveCount.Text = "0";
            lblActiveCount.Font = new Font("Segoe UI", 22F, FontStyle.Bold);
            lblActiveCount.ForeColor = emerald;
            lblActiveCount.Location = new Point(12, 100);
            lblActiveCount.AutoSize = true;

            lblPingLabel = new Label();
            lblPingLabel.Text = "BRIDGE PING";
            lblPingLabel.Font = new Font("Segoe UI", 9F, FontStyle.Bold);
            lblPingLabel.ForeColor = dimText;
            lblPingLabel.Location = new Point(12, 150);
            lblPingLabel.AutoSize = true;

            lblPingValue = new Label();
            lblPingValue.Text = "-- ms";
            lblPingValue.Font = new Font("Segoe UI", 14F, FontStyle.Bold);
            lblPingValue.ForeColor = dimText;
            lblPingValue.Location = new Point(12, 172);
            lblPingValue.AutoSize = true;

            pnlStatusCard.Controls.Add(lblPingValue);
            pnlStatusCard.Controls.Add(lblPingLabel);
            pnlStatusCard.Controls.Add(lblActiveCount);
            pnlStatusCard.Controls.Add(lblActiveLabel);
            pnlStatusCard.Controls.Add(lblShieldStatus);
            pnlStatusCard.Controls.Add(lblShieldLabel);

            pnlSidebar.Controls.Add(pnlStatusCard);
            pnlSidebar.Controls.Add(lblSeparator1);
            pnlSidebar.Controls.Add(lblLogo);

            // ===================================================================
            // MAIN CONTENT AREA
            // ===================================================================
            pnlMain = new Panel();
            pnlMain.Dock = DockStyle.Fill;
            pnlMain.BackColor = bgDeep;
            pnlMain.Padding = new Padding(16, 12, 16, 12);

            // --- TOP CONTROL BAR ---
            pnlControlBar = new Panel();
            pnlControlBar.Dock = DockStyle.Top;
            pnlControlBar.Height = 68;
            pnlControlBar.BackColor = bgPanel;
            pnlControlBar.Padding = new Padding(12, 10, 12, 10);

            btnActivateShield = new Button();
            btnActivateShield.Text = "\u25C9  ACTIVATE STEALTH SHIELD";
            btnActivateShield.Font = new Font("Segoe UI", 10F, FontStyle.Bold);
            btnActivateShield.BackColor = Color.FromArgb(0, 77, 40);
            btnActivateShield.ForeColor = emerald;
            btnActivateShield.FlatStyle = FlatStyle.Flat;
            btnActivateShield.FlatAppearance.BorderColor = emerald;
            btnActivateShield.FlatAppearance.BorderSize = 1;
            btnActivateShield.Size = new Size(280, 44);
            btnActivateShield.Location = new Point(12, 12);
            btnActivateShield.Cursor = Cursors.Hand;

            lblSlippageLabel = new Label();
            lblSlippageLabel.Text = "Slippage (Pips):";
            lblSlippageLabel.ForeColor = dimText;
            lblSlippageLabel.Font = new Font("Segoe UI", 8.5F);
            lblSlippageLabel.Location = new Point(310, 8);
            lblSlippageLabel.AutoSize = true;

            nudSlippage = new NumericUpDown();
            nudSlippage.Minimum = 0;
            nudSlippage.Maximum = 50;
            nudSlippage.Value = 3;
            nudSlippage.DecimalPlaces = 1;
            nudSlippage.Increment = 0.5M;
            nudSlippage.BackColor = bgDeep;
            nudSlippage.ForeColor = brightText;
            nudSlippage.Font = new Font("Segoe UI", 10F, FontStyle.Bold);
            nudSlippage.Location = new Point(310, 28);
            nudSlippage.Size = new Size(90, 30);
            nudSlippage.BorderStyle = BorderStyle.FixedSingle;

            lblSpreadLabel = new Label();
            lblSpreadLabel.Text = "Max Spread Filter:";
            lblSpreadLabel.ForeColor = dimText;
            lblSpreadLabel.Font = new Font("Segoe UI", 8.5F);
            lblSpreadLabel.Location = new Point(420, 8);
            lblSpreadLabel.AutoSize = true;

            nudMaxSpread = new NumericUpDown();
            nudMaxSpread.Minimum = 0;
            nudMaxSpread.Maximum = 100;
            nudMaxSpread.Value = 5;
            nudMaxSpread.DecimalPlaces = 1;
            nudMaxSpread.Increment = 0.5M;
            nudMaxSpread.BackColor = bgDeep;
            nudMaxSpread.ForeColor = brightText;
            nudMaxSpread.Font = new Font("Segoe UI", 10F, FontStyle.Bold);
            nudMaxSpread.Location = new Point(420, 28);
            nudMaxSpread.Size = new Size(90, 30);
            nudMaxSpread.BorderStyle = BorderStyle.FixedSingle;

            btnPanicFlatten = new Button();
            btnPanicFlatten.Text = "\u26A0  PANIC FLATTEN (Close All Hidden)";
            btnPanicFlatten.Font = new Font("Segoe UI", 9.5F, FontStyle.Bold);
            btnPanicFlatten.BackColor = Color.FromArgb(80, 20, 20);
            btnPanicFlatten.ForeColor = Color.FromArgb(255, 82, 82);
            btnPanicFlatten.FlatStyle = FlatStyle.Flat;
            btnPanicFlatten.FlatAppearance.BorderColor = Color.FromArgb(255, 82, 82);
            btnPanicFlatten.FlatAppearance.BorderSize = 1;
            btnPanicFlatten.Size = new Size(300, 44);
            btnPanicFlatten.Location = new Point(540, 12);
            btnPanicFlatten.Cursor = Cursors.Hand;

            btnSimulate = new Button();
            btnSimulate.Text = "\u26A1  SIMULATE BROKER STOP HUNT";
            btnSimulate.Font = new Font("Segoe UI", 9.5F, FontStyle.Bold);
            btnSimulate.BackColor = Color.FromArgb(50, 50, 10);
            btnSimulate.ForeColor = amber;
            btnSimulate.FlatStyle = FlatStyle.Flat;
            btnSimulate.FlatAppearance.BorderColor = amber;
            btnSimulate.FlatAppearance.BorderSize = 1;
            btnSimulate.Size = new Size(300, 44);
            btnSimulate.Location = new Point(860, 12);
            btnSimulate.Cursor = Cursors.Hand;

            pnlControlBar.Controls.Add(btnSimulate);
            pnlControlBar.Controls.Add(btnPanicFlatten);
            pnlControlBar.Controls.Add(nudMaxSpread);
            pnlControlBar.Controls.Add(lblSpreadLabel);
            pnlControlBar.Controls.Add(nudSlippage);
            pnlControlBar.Controls.Add(lblSlippageLabel);
            pnlControlBar.Controls.Add(btnActivateShield);

            // --- GRID AREA ---
            dgvPositions = new DataGridView();
            dgvPositions.Dock = DockStyle.Fill;
            dgvPositions.BackgroundColor = bgDeep;
            dgvPositions.GridColor = Color.FromArgb(30, 38, 50);
            dgvPositions.BorderStyle = BorderStyle.None;
            dgvPositions.CellBorderStyle = DataGridViewCellBorderStyle.SingleHorizontal;
            dgvPositions.ColumnHeadersBorderStyle = DataGridViewHeaderBorderStyle.Single;
            dgvPositions.EnableHeadersVisualStyles = false;
            dgvPositions.RowHeadersVisible = false;
            dgvPositions.AllowUserToAddRows = false;
            dgvPositions.AllowUserToDeleteRows = false;
            dgvPositions.AllowUserToResizeRows = false;
            dgvPositions.ReadOnly = true;
            dgvPositions.SelectionMode = DataGridViewSelectionMode.FullRowSelect;
            dgvPositions.MultiSelect = false;
            dgvPositions.AutoSizeColumnsMode = DataGridViewAutoSizeColumnsMode.Fill;
            dgvPositions.RowTemplate.Height = 38;
            dgvPositions.Font = new Font("Consolas", 9.5F, FontStyle.Regular);
            dgvPositions.DefaultCellStyle.BackColor = bgDeep;
            dgvPositions.DefaultCellStyle.ForeColor = emerald;
            dgvPositions.DefaultCellStyle.SelectionBackColor = Color.FromArgb(20, 35, 30);
            dgvPositions.DefaultCellStyle.SelectionForeColor = emerald;
            dgvPositions.DefaultCellStyle.Padding = new Padding(4, 0, 4, 0);
            dgvPositions.ColumnHeadersDefaultCellStyle.BackColor = bgPanel;
            dgvPositions.ColumnHeadersDefaultCellStyle.ForeColor = dimText;
            dgvPositions.ColumnHeadersDefaultCellStyle.Font = new Font("Segoe UI", 9F, FontStyle.Bold);
            dgvPositions.ColumnHeadersDefaultCellStyle.Alignment = DataGridViewContentAlignment.MiddleCenter;
            dgvPositions.ColumnHeadersDefaultCellStyle.SelectionBackColor = bgPanel;
            dgvPositions.ColumnHeadersDefaultCellStyle.SelectionForeColor = dimText;
            dgvPositions.ColumnHeadersHeight = 40;
            dgvPositions.ColumnHeadersHeightSizeMode = DataGridViewColumnHeadersHeightSizeMode.DisableResizing;

            var colTicket = new DataGridViewTextBoxColumn { Name = "Ticket", HeaderText = "TICKET ID", FillWeight = 10 };
            var colSymbol = new DataGridViewTextBoxColumn { Name = "Symbol", HeaderText = "SYMBOL", FillWeight = 10 };
            var colType = new DataGridViewTextBoxColumn { Name = "Type", HeaderText = "TYPE", FillWeight = 7 };
            var colVolume = new DataGridViewTextBoxColumn { Name = "Volume", HeaderText = "VOLUME", FillWeight = 7 };
            var colOpen = new DataGridViewTextBoxColumn { Name = "OpenPrice", HeaderText = "OPEN PRICE", FillWeight = 10 };
            var colLive = new DataGridViewTextBoxColumn { Name = "LivePrice", HeaderText = "LIVE PRICE", FillWeight = 10 };
            var colSL = new DataGridViewTextBoxColumn { Name = "HiddenSL", HeaderText = "HIDDEN SL", FillWeight = 10 };
            var colTP = new DataGridViewTextBoxColumn { Name = "HiddenTP", HeaderText = "HIDDEN TP", FillWeight = 10 };
            var colRemaining = new DataGridViewTextBoxColumn { Name = "RemainingPips", HeaderText = "PIPS TO SL", FillWeight = 10 };
            var colStatus = new DataGridViewTextBoxColumn { Name = "Status", HeaderText = "PROTECTION STATUS", FillWeight = 16 };

            dgvPositions.Columns.AddRange(colTicket, colSymbol, colType, colVolume, colOpen, colLive, colSL, colTP, colRemaining, colStatus);

            foreach (DataGridViewColumn col in dgvPositions.Columns)
            {
                col.DefaultCellStyle.Alignment = DataGridViewContentAlignment.MiddleCenter;
                col.SortMode = DataGridViewColumnSortMode.NotSortable;
            }

            // --- SPLITTER ---
            splitMain = new SplitContainer();
            splitMain.Dock = DockStyle.Fill;
            splitMain.Orientation = Orientation.Horizontal;
            splitMain.BackColor = bgDeep;
            splitMain.SplitterDistance = 480;
            splitMain.SplitterWidth = 6;
            splitMain.Panel1.BackColor = bgDeep;
            splitMain.Panel2.BackColor = bgDeep;
            splitMain.Panel1.Controls.Add(dgvPositions);
            splitMain.FixedPanel = FixedPanel.None;

            // --- LOG AREA ---
            lblLogHeader = new Label();
            lblLogHeader.Text = "  \u25B6  STEALTH EXECUTION LOG";
            lblLogHeader.Dock = DockStyle.Top;
            lblLogHeader.Height = 28;
            lblLogHeader.Font = new Font("Segoe UI", 9F, FontStyle.Bold);
            lblLogHeader.ForeColor = dimText;
            lblLogHeader.BackColor = bgPanel;
            lblLogHeader.TextAlign = ContentAlignment.MiddleLeft;

            rtbLog = new RichTextBox();
            rtbLog.Dock = DockStyle.Fill;
            rtbLog.BackColor = Color.FromArgb(8, 10, 14);
            rtbLog.ForeColor = Color.FromArgb(100, 115, 140);
            rtbLog.Font = new Font("Consolas", 9F, FontStyle.Regular);
            rtbLog.ReadOnly = true;
            rtbLog.BorderStyle = BorderStyle.None;
            rtbLog.ScrollBars = RichTextBoxScrollBars.Vertical;
            rtbLog.WordWrap = false;

            splitMain.Panel2.Controls.Add(rtbLog);
            splitMain.Panel2.Controls.Add(lblLogHeader);

            pnlMain.Controls.Add(splitMain);
            pnlMain.Controls.Add(pnlControlBar);

            // === Assemble Form ===
            Controls.Add(pnlMain);
            Controls.Add(pnlSidebar);

            ResumeLayout(false);
            PerformLayout();
        }

        #endregion

        private Panel pnlSidebar;
        private Label lblLogo;
        private Label lblSeparator1;
        private Panel pnlStatusCard;
        private Label lblShieldLabel;
        private Label lblShieldStatus;
        private Label lblActiveLabel;
        private Label lblActiveCount;
        private Label lblPingLabel;
        private Label lblPingValue;

        private Panel pnlMain;
        private Panel pnlControlBar;
        private Button btnActivateShield;
        private Label lblSlippageLabel;
        private NumericUpDown nudSlippage;
        private Label lblSpreadLabel;
        private NumericUpDown nudMaxSpread;
        private Button btnPanicFlatten;
        private Button btnSimulate;

        private SplitContainer splitMain;
        private DataGridView dgvPositions;
        private Label lblLogHeader;
        private RichTextBox rtbLog;
    }
}
