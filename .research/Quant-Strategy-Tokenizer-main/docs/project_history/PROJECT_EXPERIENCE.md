# Quant Strategy Tokenizer 项目经历
*EN: Quant Strategy Tokenizer Project Experience*

<p align="center">
  <img src="assets/performance-90d.png" alt="90-day strategy performance snapshot" width="100%">
</p>

*90 天实盘表现截图（项目背景材料）*
*EN: 90-day live performance snapshot used as project background context.*

你们好啊，我将在这里添加一些跟项目有关的背景和经历
*EN: Hello, I will add some project-related background and experience here.*

起因是主包历时六个月写了一个量化策略，这个策略本身是一个针对加密货币的多标的均值回归系统
*EN: The starting point was that I spent six months building a quantitative strategy: a multi-asset mean-reversion system for cryptocurrencies.*

在回测和实盘的前两个月，这个系统展现了极强的盈利能力
*EN: During backtesting and the first two months of live trading, the system showed very strong profitability.*

三年的回测收益率一度来到700%，实盘两个月收益来到300%
*EN: At one point, the three-year backtest return reached 700%, and the live return reached 300% over two months.*

但是在btc 2026年五月五日的单边震荡行情中亏了一波大的，利润几乎全部回吐
*EN: However, during BTC's one-sided choppy move on May 5, 2026, the strategy suffered a large loss and gave back almost all profits.*

更惨的是由于主包的盲目自信，还导致了策略的盘外损失
*EN: Worse, my own overconfidence also caused losses outside the strategy itself.*

事后分析得出结论，这个策略的收益主要来自风险溢价
*EN: The post-mortem conclusion was that the strategy's returns mainly came from risk premium.*

它之所以在回测或平时能赚钱，是因为它使用了分批加仓（Grid）
*EN: It made money in backtests and normal markets because it used staged adding, or a grid-like structure.*

这在数学上等同于做空波动率（Short Volatility / Short Gamma）
*EN: Mathematically, this is equivalent to being short volatility or short gamma.*

它通过牺牲尾部风险，换取了平时的高胜率
*EN: It exchanged tail-risk exposure for a high win rate in ordinary conditions.*



这个策略本身的信号是没有α收益的，但是却能在特定的行情内盈利
*EN: The strategy's signal itself did not have alpha, yet it could profit in specific market regimes.*

辩证的说，整个系统本质上就是一个巨大的持仓策略而不是一个信号识别器
*EN: Dialectically speaking, the whole system was essentially a large position-management strategy rather than a signal detector.*

但是如果一个持仓系统就能够在没有边际正期望的情况下盈利
*EN: But if a position-management system can profit without positive marginal expectancy,*

那么是否意味着我们能够将场景限制在一个非常理想的条件下进行交易呢?
*EN: does that mean we can restrict trading to a very ideal set of conditions?*

于是整个策略围绕均值回归的假设做出了很多尝试
*EN: So the entire strategy went through many experiments around the mean-reversion assumption.*

首先就是通过层层过滤筛选掉了在量价关系上不适合回归的标的
*EN: First, layered filters removed instruments whose price-volume behavior was not suitable for reversion.*

然后根据btc的带动效应有设计了vwap加仓系统对加仓进行优化
*EN: Then, based on BTC's leadership effect, a VWAP-based add system was designed to optimize position adding.*

总之，就是策略本身已经将所有不满足均值回归这一假设的regime全都剔除了
*EN: In short, the strategy had already removed regimes that did not satisfy the mean-reversion assumption.*

但是由于策略收益来自风险溢价，所以在市场环境下如果要产生收益就必须保留在风险中
*EN: But because the returns came from risk premium, the system had to remain exposed to risk in order to earn returns.*

于是这个策略最害怕的行情出现了，震荡上涨/下跌
*EN: Then the regime this strategy feared most appeared: a choppy one-sided rise or fall.*



首先，震荡行情会导致策略的风控系统失效，因为震荡实际上属于均值回归的盈利区间
*EN: First, choppy markets can make the risk system fail, because chop itself is normally the profitable zone for mean reversion.*

然而震荡上涨和下跌是单边的，即回调软弱无力
*EN: However, choppy rises and falls are one-sided: the pullbacks are weak and ineffective.*

而一个策略的最大仓位又是有限的，所以当仓位过载，而趋势仍然继续时就会出现仓位整体的均价被强行拉高
*EN: Because any strategy has a finite maximum position, once the position is overloaded and the trend continues, the overall average entry price is forcibly dragged upward.*

没有仓位能够平仓导致整个账户的平均开仓成本没办法更新，于是进入亏损周期
*EN: When no positions can be closed, the account's average entry cost cannot refresh, and the system enters a loss cycle.*

我解决这个问题的方法是增大最大开仓数量来保证仓位的流动性，这样就能在震荡上升中不断的开出新仓位从而拉平平均开仓价格
*EN: My solution was to increase the maximum number of entries to preserve position liquidity, allowing new positions to keep opening during a choppy rise and smooth the average entry price.*

这就面临两个问题，首先是开仓数量会导致仓位变小，并导致小资金无法正常完成交易周期
*EN: This creates two problems. First, more entries make each position smaller and can prevent small accounts from completing a normal trading cycle.*

二是如果我们将震荡上升的斜率和开仓速度的斜率做成比值，就不难发现，整个系统在最大开仓数量之下是线性的
*EN: Second, if we compare the slope of the choppy rise with the slope of entry speed, it becomes clear that the system is linear below the maximum number of entries.*

这种线性我称之为仓位流动性，但是仓位流动性的线性是有界的
*EN: I call this linearity position liquidity, but the linearity of position liquidity is bounded.*

但是在最大开仓数之外，这种线性就被破坏了，整个流动性就坍缩成了一个（最大开仓/x）的函数
*EN: Beyond the maximum number of entries, this linearity breaks, and the whole liquidity structure collapses into a function like max_entries / x.*

根据初中知识，这个函数的特性之一就是整个k值在接近x轴时趋于平缓
*EN: By basic middle-school math, one property of this function is that its slope flattens as it approaches the x-axis.*

于是系统的调节能力不会数学上直接归零，但会变得小到没有实际意义
*EN: So the system's adjustment ability does not mathematically fall directly to zero, but it becomes too small to matter in practice.*

更近一步的，在加仓种同样存在这种问题，加仓如果可以一直加仓那么这个系统也会是线性的
*EN: Going further, the same problem exists in position adding: if adding could continue indefinitely, the system would also remain linear.*

但是这种线性依然会在加仓达到最大值时被破坏，这是马丁格尔策略的结论，而我通过将仓位缩小进一步获得了一个非常非常深的加仓极限
*EN: But that linearity still breaks when adding reaches its maximum. This is the conclusion of martingale-style systems, and by shrinking position size I obtained a much deeper add limit.*

所以这个策略的极限是可以通过正交化后的风险约束求出来的。
*EN: Therefore, the limits of this strategy can be derived through orthogonalized risk constraints.*

继续在这条上添加规律实际上是可以进一步提升收益稳定性和系统边界
*EN: Continuing to add rules along this path could further improve return stability and expand the system boundary.*

最终一定可以以一种非常暴力的方式完成一个稳定盈利的机器
*EN: Eventually, it would likely be possible to build a stable profit machine in a very forceful way.*

但是如果我继续在这上面花时间，将十分不利于学习新东西
*EN: But if I kept spending time on this, it would be very bad for learning new things.*

并且这个系统已经太笨重了，丝毫没有任何美感可言
*EN: Also, the system had become too heavy and had almost no aesthetic quality left.*

在数学上和设计哲学上都缺乏美感，让人没有继续下去的欲望，通俗的说就是屎上雕花
*EN: It lacked elegance both mathematically and philosophically, which made me lose the desire to continue. In plain terms, it felt like carving flowers on something fundamentally ugly.*

于是主包将整个程序肢解之后开源，将我在此期间的设计一般化
*EN: So I decomposed the whole program, open-sourced it, and generalized the designs I developed during the process.*

一方面这些设计在完成度上比较高，很多也较为实用
*EN: On one hand, many of these designs are fairly complete and practically useful.*

另一方面是主包早年通过开源渠道获得过很多帮助，在这里小小回馈一下社区
*EN: On the other hand, I benefited a lot from open source earlier, so this is a small way to give back to the community.*



整个Quant Strategy Tokenizer项目都是围绕agent设计的，意思是将策略肢解成agent喜闻乐见的小尸块: "token"
*EN: The whole Quant Strategy Tokenizer project is designed around agents: it breaks strategies into small agent-friendly pieces, or tokens.*

旨在能让你的agent快速的学会量化交易策略的编写和直接调用我已经造出来的轮子
*EN: The goal is to let your agent quickly learn how to build quantitative trading strategies and directly call the reusable components I have already built.*

这么做有两个好处，首先是你可以让你的agent 例如 codex或Claude code 乃至 deepseek 直接替你写东西
*EN: This has two benefits. First, you can let agents such as Codex, Claude Code, or even DeepSeek write things directly for you.*

可以使用公共api或是你的神秘数据库，减少在代码实现到部分回测上减少泄密风险，并且无需注册和调试复杂低效的量化交易平台
*EN: You can use public APIs or your own private database, reduce leakage risk during implementation and partial backtesting, and avoid registering for and debugging complex, inefficient quant platforms.*

我认为，这些拿着开源技术收费的公司在道义上是说不过去的
*EN: In my view, companies that charge for wrapping open-source technology have a weak moral position.*

另外，你自己也可以有针对性的对模块进行修改，定制自己的参数乃至实现方法
*EN: Second, you can modify modules directly, customizing your own parameters or even implementation methods.*

之后我也会继续在这里跟新新的模块来扩展整个项目的边界，也欢迎同行加入
*EN: I will also keep adding new modules here to expand the project boundary, and peers are welcome to join.*

核心思想其实很简单，就是面向readme的编程（虽然我在哲学上不是很认可），将轮子扔给ai去写
*EN: The core idea is simple: README-oriented programming, although I do not fully endorse it philosophically, where reusable components are handed to AI to work with.*

代码中的错误和不成熟的地方，欢迎斧正
*EN: Corrections are welcome for any errors or immature parts of the code.*



原策略时间线
*EN: Original Strategy Timeline*

2025-08

KDJ / 技术指标胜率研究
*EN: KDJ / technical-indicator win-rate research*

&#x20;       ↓

2025-09

横截面机器学习、多空双通道、BTC tilt、路径 SL/TP
*EN: Cross-sectional machine learning, dual long/short channels, BTC tilt, path-based SL/TP*

&#x20;       ↓

2025-10

BTC regime、Markov 状态、NO-TRADE 门控
*EN: BTC regime, Markov state, NO-TRADE gating*

&#x20;       ↓

2025-11

15m EMA/ATR 均值回归实盘机器人诞生
*EN: The 15m EMA/ATR live mean-reversion bot was born*

&#x20;       ↓

2025-11 下旬
*EN: Late 2025-11*

DTF / HTF 多周期方向、SpikeGuard、MRQ-A
*EN: DTF / HTF multi-timeframe direction, SpikeGuard, MRQ-A*

&#x20;       ↓

2025-12

slot budget、market freeze、openOrders cache、CircuitBreaker

&#x20;       ↓

2026-01

市场冻结确认、冻结期撤风险挂单、恢复 ADD
*EN: Market-freeze confirmation, risk-order cancellation during freeze, ADD restoration*

&#x20;       ↓

2026-02

冻结 observe 观察期
*EN: Freeze observe period*

&#x20;       ↓

2026-03

公域/私域分化、state\_path、规模化参数
*EN: Public/private instance separation, state_path, scaled parameters*

&#x20;       ↓

2026-05

BTC/Symbol VWAP、BTC-neutral、entry vote、degraded、451、unknown flat、结构化审计
*EN: BTC/Symbol VWAP, BTC-neutral, entry vote, degraded mode, 451 handling, unknown flat, structured audit*



