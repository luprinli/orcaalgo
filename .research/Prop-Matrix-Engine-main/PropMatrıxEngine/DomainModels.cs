namespace PropMatrixEngine;

public enum Mt5ConnectionMode
{
    SignalR,
    Simulation
}

public enum AccountConnectionStatus
{
    Disconnected,
    Connecting,
    Connected,
    Degraded,
    Killing,
    Killed,
    Failed
}

public enum ChartLineKind
{
    HorizontalPrice,
    TrendLine
}

public enum OrderSide
{
    Buy,
    Sell
}

public enum PendingOrderKind
{
    BuyLimit,
    SellLimit,
    BuyStop,
    SellStop
}

public enum TradeCommandType
{
    CloseAllPositions,
    CancelAllPendingOrders,
    PlacePendingOrder,
    DisableTrading
}

public sealed record AccountEndpoint(
    string AccountId,
    string DisplayName,
    string Server,
    string HubUrl,
    Mt5ConnectionMode Mode);

public sealed record AccountSnapshot(
    string AccountId,
    string DisplayName,
    string Server,
    long Login,
    decimal Balance,
    decimal Equity,
    decimal FloatingProfitLoss,
    decimal Margin,
    decimal FreeMargin,
    decimal Exposure,
    DateTimeOffset ServerTime,
    AccountConnectionStatus Status);

public sealed record ChartLineSignal(
    string SourceAccountId,
    string Symbol,
    ChartLineKind Kind,
    decimal Price,
    DateTimeOffset? ExpiresAt,
    string Label,
    DateTimeOffset ObservedAt);

public sealed record PendingOrderRequest(
    string TargetAccountId,
    string Symbol,
    PendingOrderKind OrderKind,
    decimal Price,
    decimal VolumeLots,
    string CorrelationId,
    string Comment);

public sealed record TradeCommand(
    TradeCommandType Type,
    string CorrelationId,
    PendingOrderRequest? PendingOrder,
    DateTimeOffset CreatedAt)
{
    public static TradeCommand KillSwitch(string correlationId) =>
        new(TradeCommandType.CloseAllPositions, correlationId, null, DateTimeOffset.UtcNow);

    public static TradeCommand CancelPending(string correlationId) =>
        new(TradeCommandType.CancelAllPendingOrders, correlationId, null, DateTimeOffset.UtcNow);

    public static TradeCommand DisableTrading(string correlationId) =>
        new(TradeCommandType.DisableTrading, correlationId, null, DateTimeOffset.UtcNow);

    public static TradeCommand PlacePending(PendingOrderRequest order) =>
        new(TradeCommandType.PlacePendingOrder, order.CorrelationId, order, DateTimeOffset.UtcNow);
}

public sealed record TradeCommandResult(
    string AccountId,
    string CorrelationId,
    TradeCommandType Type,
    bool Success,
    string Message,
    TimeSpan Elapsed);

public sealed record GlobalRiskSnapshot(
    decimal TotalBalance,
    decimal TotalEquity,
    decimal TotalFloatingProfitLoss,
    decimal DailyPeakEquity,
    decimal DrawdownPercent,
    decimal DrawdownLimitPercent,
    bool IsArmed,
    bool IsKillSwitchRequired,
    string Reason);
