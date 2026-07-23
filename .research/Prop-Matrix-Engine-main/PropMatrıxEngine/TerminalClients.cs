using Microsoft.AspNetCore.SignalR.Client;
using System.Diagnostics;

namespace PropMatrixEngine;

public interface IMt5TerminalClient : IAsyncDisposable
{
    AccountEndpoint Endpoint { get; }
    AccountSnapshot? LatestSnapshot { get; }
    AccountConnectionStatus Status { get; }

    event EventHandler<AccountSnapshot>? SnapshotReceived;
    event EventHandler<ChartLineSignal>? ChartLineReceived;
    event EventHandler<TradeCommandResult>? CommandCompleted;
    event EventHandler<AccountConnectionStatus>? StatusChanged;

    Task ConnectAsync(CancellationToken cancellationToken);
    Task DisconnectAsync(CancellationToken cancellationToken);
    Task<TradeCommandResult> SendCommandAsync(TradeCommand command, CancellationToken cancellationToken);
}

public sealed class SignalRMt5TerminalClient : IMt5TerminalClient
{
    private HubConnection? _connection;

    public SignalRMt5TerminalClient(AccountEndpoint endpoint)
    {
        Endpoint = endpoint;
    }

    public AccountEndpoint Endpoint { get; }
    public AccountSnapshot? LatestSnapshot { get; private set; }
    public AccountConnectionStatus Status { get; private set; } = AccountConnectionStatus.Disconnected;

    public event EventHandler<AccountSnapshot>? SnapshotReceived;
    public event EventHandler<ChartLineSignal>? ChartLineReceived;
    public event EventHandler<TradeCommandResult>? CommandCompleted;
    public event EventHandler<AccountConnectionStatus>? StatusChanged;

    public async Task ConnectAsync(CancellationToken cancellationToken)
    {
        SetStatus(AccountConnectionStatus.Connecting);

        _connection = new HubConnectionBuilder()
            .WithUrl(Endpoint.HubUrl)
            .WithAutomaticReconnect()
            .Build();

        _connection.On<AccountSnapshot>("AccountSnapshot", snapshot =>
        {
            LatestSnapshot = snapshot with { Status = AccountConnectionStatus.Connected };
            SnapshotReceived?.Invoke(this, LatestSnapshot);
        });

        _connection.On<ChartLineSignal>("ChartLineChanged", signal =>
            ChartLineReceived?.Invoke(this, signal));

        _connection.On<TradeCommandResult>("TradeCommandResult", result =>
            CommandCompleted?.Invoke(this, result));

        _connection.Reconnecting += _ =>
        {
            SetStatus(AccountConnectionStatus.Degraded);
            return Task.CompletedTask;
        };

        _connection.Reconnected += _ =>
        {
            SetStatus(AccountConnectionStatus.Connected);
            return Task.CompletedTask;
        };

        _connection.Closed += _ =>
        {
            SetStatus(AccountConnectionStatus.Disconnected);
            return Task.CompletedTask;
        };

        await _connection.StartAsync(cancellationToken).ConfigureAwait(false);
        await _connection.InvokeAsync("RegisterTerminal", Endpoint.AccountId, cancellationToken).ConfigureAwait(false);
        SetStatus(AccountConnectionStatus.Connected);
    }

    public async Task DisconnectAsync(CancellationToken cancellationToken)
    {
        if (_connection is not null)
        {
            await _connection.StopAsync(cancellationToken).ConfigureAwait(false);
            await _connection.DisposeAsync().ConfigureAwait(false);
            _connection = null;
        }

        SetStatus(AccountConnectionStatus.Disconnected);
    }

    public async Task<TradeCommandResult> SendCommandAsync(TradeCommand command, CancellationToken cancellationToken)
    {
        if (_connection is null)
        {
            return Failed(command, "SignalR hub is not connected.");
        }

        var stopwatch = Stopwatch.StartNew();
        try
        {
            var result = await _connection.InvokeAsync<TradeCommandResult>(
                "ExecuteCommand",
                Endpoint.AccountId,
                command,
                cancellationToken).ConfigureAwait(false);

            stopwatch.Stop();
            result = result with { Elapsed = stopwatch.Elapsed };
            CommandCompleted?.Invoke(this, result);
            return result;
        }
        catch (Exception ex)
        {
            stopwatch.Stop();
            var result = Failed(command, ex.Message) with { Elapsed = stopwatch.Elapsed };
            CommandCompleted?.Invoke(this, result);
            return result;
        }
    }

    public async ValueTask DisposeAsync()
    {
        if (_connection is not null)
        {
            await _connection.DisposeAsync().ConfigureAwait(false);
        }
    }

    private TradeCommandResult Failed(TradeCommand command, string message) =>
        new(Endpoint.AccountId, command.CorrelationId, command.Type, false, message, TimeSpan.Zero);

    private void SetStatus(AccountConnectionStatus status)
    {
        Status = status;
        StatusChanged?.Invoke(this, status);
    }
}

public sealed class SimulatedMt5TerminalClient : IMt5TerminalClient
{
    private readonly decimal _baseBalance;
    private readonly long _login;
    private CancellationTokenSource? _loopCts;
    private Task? _loopTask;
    private decimal _floatingProfitLoss;

    public SimulatedMt5TerminalClient(AccountEndpoint endpoint, long login, decimal baseBalance)
    {
        Endpoint = endpoint;
        _login = login;
        _baseBalance = baseBalance;
    }

    public AccountEndpoint Endpoint { get; }
    public AccountSnapshot? LatestSnapshot { get; private set; }
    public AccountConnectionStatus Status { get; private set; } = AccountConnectionStatus.Disconnected;

    public event EventHandler<AccountSnapshot>? SnapshotReceived;
    public event EventHandler<ChartLineSignal>? ChartLineReceived;
    public event EventHandler<TradeCommandResult>? CommandCompleted;
    public event EventHandler<AccountConnectionStatus>? StatusChanged;

    public Task ConnectAsync(CancellationToken cancellationToken)
    {
        SetStatus(AccountConnectionStatus.Connected);
        _loopCts = CancellationTokenSource.CreateLinkedTokenSource(cancellationToken);
        _loopTask = Task.Run(() => PublishLoopAsync(_loopCts.Token), CancellationToken.None);
        return Task.CompletedTask;
    }

    public async Task DisconnectAsync(CancellationToken cancellationToken)
    {
        if (_loopCts is not null)
        {
            await _loopCts.CancelAsync().ConfigureAwait(false);
            _loopCts.Dispose();
            _loopCts = null;
        }

        if (_loopTask is not null)
        {
            await _loopTask.WaitAsync(TimeSpan.FromSeconds(2), cancellationToken).ConfigureAwait(false);
            _loopTask = null;
        }

        SetStatus(AccountConnectionStatus.Disconnected);
    }

    public async Task<TradeCommandResult> SendCommandAsync(TradeCommand command, CancellationToken cancellationToken)
    {
        var stopwatch = Stopwatch.StartNew();
        SetStatus(command.Type == TradeCommandType.CloseAllPositions ? AccountConnectionStatus.Killing : Status);
        await Task.Delay(Random.Shared.Next(3, 18), cancellationToken).ConfigureAwait(false);

        if (command.Type is TradeCommandType.CloseAllPositions or TradeCommandType.CancelAllPendingOrders)
        {
            _floatingProfitLoss = 0;
            PublishSnapshot(AccountConnectionStatus.Killed);
        }

        var result = new TradeCommandResult(
            Endpoint.AccountId,
            command.CorrelationId,
            command.Type,
            true,
            $"{command.Type} accepted by {Endpoint.DisplayName}.",
            stopwatch.Elapsed);

        CommandCompleted?.Invoke(this, result);
        return result;
    }

    public async ValueTask DisposeAsync()
    {
        if (_loopCts is not null)
        {
            await _loopCts.CancelAsync().ConfigureAwait(false);
            _loopCts.Dispose();
        }
    }

    private async Task PublishLoopAsync(CancellationToken cancellationToken)
    {
        var ticks = 0;
        while (!cancellationToken.IsCancellationRequested)
        {
            PublishSnapshot(AccountConnectionStatus.Connected);
            ticks++;

            if (ticks % 15 == 0)
            {
                ChartLineReceived?.Invoke(this, new ChartLineSignal(
                    Endpoint.AccountId,
                    "EURUSD",
                    ChartLineKind.HorizontalPrice,
                    decimal.Round(1.07m + (decimal)Random.Shared.NextDouble() / 100m, 5),
                    DateTimeOffset.UtcNow.AddHours(6),
                    ticks % 30 == 0 ? "BUY LIMIT liquidity line" : "SELL LIMIT hedge line",
                    DateTimeOffset.UtcNow));
            }

            await Task.Delay(800, cancellationToken).ConfigureAwait(false);
        }
    }

    private void PublishSnapshot(AccountConnectionStatus status)
    {
        var drift = (decimal)(Random.Shared.NextDouble() - 0.5d) * 220m;
        _floatingProfitLoss = decimal.Round(_floatingProfitLoss + drift, 2);
        var equity = _baseBalance + _floatingProfitLoss;

        LatestSnapshot = new AccountSnapshot(
            Endpoint.AccountId,
            Endpoint.DisplayName,
            Endpoint.Server,
            _login,
            _baseBalance,
            equity,
            _floatingProfitLoss,
            Math.Max(0, equity * 0.12m),
            Math.Max(0, equity * 0.88m),
            Math.Abs(_floatingProfitLoss) * 3.4m,
            DateTimeOffset.Now,
            status);

        SetStatus(status);
        SnapshotReceived?.Invoke(this, LatestSnapshot);
    }

    private void SetStatus(AccountConnectionStatus status)
    {
        Status = status;
        StatusChanged?.Invoke(this, status);
    }
}
