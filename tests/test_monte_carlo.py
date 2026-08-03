from __future__ import annotations

import numpy as np

from orca.backtest.monte_carlo import resample_actual_trades


class TestResampleBlockBootstrap:
    def test_basic_resample(self):
        pnls = [1.0, -0.5, 2.0, -0.3, 1.5, -0.8, 0.7, -0.2, 0.9, -0.6]
        result = resample_actual_trades(pnls, 20, block_len=3)
        assert len(result) == 20
        for v in result:
            assert v in pnls

    def test_block_preserves_sequence(self):
        pnls = list(range(100))
        result = resample_actual_trades(pnls, 12, block_len=3)
        assert len(result) == 12
        blocks_found = False
        for i in range(len(result) - 3):
            a, b, c = result[i], result[i+1], result[i+2]
            if b == a + 1 and c == a + 2:
                blocks_found = True
                break
        assert blocks_found, "Block bootstrap should preserve contiguous sequences"

    def test_block_len_one(self):
        pnls = [1.0, -0.5, 2.0]
        result = resample_actual_trades(pnls, 10, block_len=1)
        assert len(result) == 10
        for v in result:
            assert v in pnls

    def test_block_len_exceeds_data(self):
        pnls = [1.0, -0.5, 2.0, -0.3, 1.5]
        result = resample_actual_trades(pnls, 10, block_len=10)
        assert len(result) == 10
        assert len(result) > 0

    def test_empty_pnls(self):
        result = resample_actual_trades([], 10, block_len=3)
        assert len(result) == 0

    def test_zero_trades_requested(self):
        pnls = [1.0, -0.5, 2.0]
        result = resample_actual_trades(pnls, 0, block_len=3)
        assert len(result) == 3  # defaults to len(actual_pnls)

    def test_block_len_seven_default(self):
        pnls = np.random.normal(0.001, 0.01, 200).tolist()
        result = resample_actual_trades(pnls, 50)
        assert len(result) == 50

    def test_wrap_around(self):
        pnls = [10.0, 20.0, 30.0, 40.0, 50.0]
        result = resample_actual_trades(pnls, 15, block_len=3)
        assert len(result) == 15
        for v in result:
            assert v in pnls

    def test_single_value_input(self):
        pnls = [0.5]
        result = resample_actual_trades(pnls, 5, block_len=3)
        assert len(result) == 5
        for v in result:
            assert v in pnls
