using System.Collections.Concurrent;
using System.Diagnostics;

namespace PropMatrixEngine;

public sealed class AccountRegistry : IAsyncDisposable
{
    private readonly ConcurrentDictionary<string, IMt5TerminalClient> _clients = new();
    private readonly ConcurrentDictionary<string, AccountSnapshot> _snapshots = new();

    public event EventHandler<AccountSnapshot>? SnapshotReceived;
    public event EventHandler<ChartLineSignal>? ChartLineReceived;
    public event EventHandler<TradeCommandResult>? CommandCompleted;

    public IReadOnlyCollection<AccountSnapshot> Snapshots =>
        _snapshots.Values.OrderBy(snapshot => snapshot.DisplayName).ToArray();

    public IReadOnlyCollection<IMt5TerminalClient> Clients =>
        _clients.Values.OrderBy(client => client.Endpoint.DisplayName).ToArray();

    public async Task AddAndConnectAsync(AccountEndpoint endpoint, CancellationToken cancellationToken)
    {
        if (_clients.ContainsKey(endpoint.AccountId))
        {
            throw new InvalidOperationException($"Account '{endpoint.DisplayName}' is already registered.");
        }

        var client = CreateClient(endpoint);
        Attach(client);

        if (!_clients.TryAdd(endpoint.AccountId, client))
        {
            await client.DisposeAsync().ConfigureAwait(false);
            throw new InvalidOperationException($"Account '{endpoint.DisplayName}' is already registered.");
        }

        try
        {
            await client.ConnectAsync(cancellationToken).ConfigureAwait(false);
        }
        catch
        {
            _clients.TryRemove(endpoint.AccountId, out _);
            Detach(client);
            await client.DisposeAsync().ConfigureAwait(false);
            throw;
        }
    }

    public async Task<IReadOnlyCollection<TradeCommandResult>> ExecuteGlobalKillSwitchAsync(CancellationToken cancellationToken)
    {
        var stopwatch = Stopwatch.StartNew();
        var correlationId = $"KILL-{DateTimeOffset.UtcNow:yyyyMMddHHmmssfff}-{Guid.NewGuid():N}";
        var tasks = Clients.Select(client => ExecuteKillSequenceAsync(client, correlationId, cancellationToken)).ToArray();
        var results = await Task.WhenAll(tasks).ConfigureAwait(false);
        stopwatch.Stop();

        var flattened = results.SelectMany(result => result).ToArray();
        CommandCompleted?.Invoke(this, new TradeCommandResult(
            "GLOBAL",
            correlationId,
            TradeCommandType.DisableTrading,
            flattened.All(result => result.Success),
            $"Global kill-switch dispatched to {_clients.Count} terminals in {stopwatch.ElapsedMilliseconds} ms.",
            stopwatch.Elapsed));

        return flattened;
    }

    public async Task<IReadOnlyCollection<TradeCommandResult>> CopyChartLineAsPendingOrdersAsync(
        ChartLineSignal signal,
        decimal defaultVolumeLots,
        CancellationToken cancellationToken)
    {
        var targets = Clients
            .Where(client => client.Endpoint.AccountId != signal.SourceAccountId)
            .Where(client => client.Status is AccountConnectionStatus.Connected or AccountConnectionStatus.Degraded)
            .ToArray();

        var tasks = targets.Select(client =>
        {
            var request = ChartLineCopyEngine.CreatePendingOrder(signal, client.Endpoint.AccountId, defaultVolumeLots);
            return client.SendCommandAsync(TradeCommand.PlacePending(request), cancellationToken);
        });

        return await Task.WhenAll(tasks).ConfigureAwait(false);
    }

    public async ValueTask DisposeAsync()
    {
        foreach (var client in Clients)
        {
            Detach(client);
            await client.DisposeAsync().ConfigureAwait(false);
        }
    }

    private static IMt5TerminalClient CreateClient(AccountEndpoint endpoint)
    {
        if (endpoint.Mode == Mt5ConnectionMode.SignalR)
        {
            return new SignalRMt5TerminalClient(endpoint);
        }

        var login = long.TryParse(endpoint.AccountId, out var parsedLogin)
            ? parsedLogin
            : Random.Shared.NextInt64(10_000_000, 99_999_999);

        return new SimulatedMt5TerminalClient(
            endpoint,
            login,
            Random.Shared.Next(75_000, 250_000));
    }

    private static async Task<IReadOnlyCollection<TradeCommandResult>> ExecuteKillSequenceAsync(
        IMt5TerminalClient client,
        string correlationId,
        CancellationToken cancellationToken)
    {
        var commands = new[]
        {
            TradeCommand.KillSwitch(correlationId),
            TradeCommand.CancelPending(correlationId),
            TradeCommand.DisableTrading(correlationId)
        };

        var results = new List<TradeCommandResult>(commands.Length);
        foreach (var command in commands)
        {
            results.Add(await client.SendCommandAsync(command, cancellationToken).ConfigureAwait(false));
        }

        return results;
    }

    private void Attach(IMt5TerminalClient client)
    {
        client.SnapshotReceived += HandleSnapshotReceived;
        client.ChartLineReceived += HandleChartLineReceived;
        client.CommandCompleted += HandleCommandCompleted;
    }

    private void Detach(IMt5TerminalClient client)
    {
        client.SnapshotReceived -= HandleSnapshotReceived;
        client.ChartLineReceived -= HandleChartLineReceived;
        client.CommandCompleted -= HandleCommandCompleted;
    }

    private void HandleSnapshotReceived(object? sender, AccountSnapshot snapshot)
    {
        _snapshots[snapshot.AccountId] = snapshot;
        SnapshotReceived?.Invoke(this, snapshot);
    }

    private void HandleChartLineReceived(object? sender, ChartLineSignal signal) =>
        ChartLineReceived?.Invoke(this, signal);

    private void HandleCommandCompleted(object? sender, TradeCommandResult result) =>
        CommandCompleted?.Invoke(this, result);
}

public sealed class RiskEngine
{
    private decimal _dailyPeakEquity;
    private bool _triggered;

    public RiskEngine(decimal drawdownLimitPercent, decimal preBreachBufferPercent)
    {
        DrawdownLimitPercent = drawdownLimitPercent;
        PreBreachBufferPercent = preBreachBufferPercent;
    }

    public decimal DrawdownLimitPercent { get; set; }
    public decimal PreBreachBufferPercent { get; set; }
    public bool IsArmed { get; private set; } = true;

    public event EventHandler<GlobalRiskSnapshot>? KillSwitchRequired;

    public void Arm()
    {
        _triggered = false;
        IsArmed = true;
    }

    public void Disarm() => IsArmed = false;

    public GlobalRiskSnapshot Evaluate(IEnumerable<AccountSnapshot> accountSnapshots)
    {
        var snapshots = accountSnapshots.ToArray();
        var totalBalance = snapshots.Sum(snapshot => snapshot.Balance);
        var totalEquity = snapshots.Sum(snapshot => snapshot.Equity);
        var totalFloatingProfitLoss = snapshots.Sum(snapshot => snapshot.FloatingProfitLoss);

        if (_dailyPeakEquity <= 0 || totalEquity > _dailyPeakEquity)
        {
            _dailyPeakEquity = totalEquity;
        }

        var drawdownPercent = _dailyPeakEquity <= 0
            ? 0
            : decimal.Round((_dailyPeakEquity - totalEquity) / _dailyPeakEquity * 100m, 2);

        var triggerAt = Math.Max(0, DrawdownLimitPercent - PreBreachBufferPercent);
        var shouldTrigger = IsArmed && !_triggered && snapshots.Length > 0 && drawdownPercent >= triggerAt;
        var risk = new GlobalRiskSnapshot(
            totalBalance,
            totalEquity,
            totalFloatingProfitLoss,
            _dailyPeakEquity,
            drawdownPercent,
            DrawdownLimitPercent,
            IsArmed,
            shouldTrigger,
            shouldTrigger
                ? $"Global drawdown {drawdownPercent:N2}% reached pre-breach threshold {triggerAt:N2}%."
                : "Within configured drawdown envelope.");

        if (shouldTrigger)
        {
            _triggered = true;
            KillSwitchRequired?.Invoke(this, risk);
        }

        return risk;
    }
}

public static class ChartLineCopyEngine
{
    public static PendingOrderRequest CreatePendingOrder(
        ChartLineSignal signal,
        string targetAccountId,
        decimal defaultVolumeLots)
    {
        var orderKind = ResolveOrderKind(signal.Label);
        return new PendingOrderRequest(
            targetAccountId,
            signal.Symbol,
            orderKind,
            signal.Price,
            defaultVolumeLots,
            $"COPY-{signal.ObservedAt:yyyyMMddHHmmssfff}-{Guid.NewGuid():N}",
            $"Replicated chart line from {signal.SourceAccountId}: {signal.Label}");
    }

    private static PendingOrderKind ResolveOrderKind(string label)
    {
        var normalized = label.ToUpperInvariant();

        if (normalized.Contains("SELL STOP", StringComparison.Ordinal))
        {
            return PendingOrderKind.SellStop;
        }

        if (normalized.Contains("BUY STOP", StringComparison.Ordinal))
        {
            return PendingOrderKind.BuyStop;
        }

        if (normalized.Contains("SELL", StringComparison.Ordinal))
        {
            return PendingOrderKind.SellLimit;
        }

        return PendingOrderKind.BuyLimit;
    }
}
