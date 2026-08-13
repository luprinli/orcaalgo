"""NYSE holiday calendar for data pipeline date filtering.

Provides a function to check whether a given date is a US market holiday
(NYSE closure). Covers major federal/observed holidays from 2020-2030.
"""

from __future__ import annotations

from datetime import date


def _year_holidays(year: int) -> list[date]:
    """Return NYSE holiday dates for a given year."""
    return [
        date(year, 1, 1),   # New Year's Day
        _nth_weekday(year, 1, 1, 3),   # MLK Day (3rd Monday of January)
        _nth_weekday(year, 2, 1, 3),   # Presidents Day (3rd Monday of February)
        _last_monday(year, 5),          # Memorial Day (last Monday of May)
        _juneteenth(year),              # Juneteenth
        date(year, 7, 4),               # Independence Day
        _nth_weekday(year, 9, 1, 1),   # Labor Day (1st Monday of September)
        _nth_weekday(year, 11, 4, 4),  # Thanksgiving (4th Thursday of November)
        date(year, 12, 25),             # Christmas Day
        _good_friday(year),             # Good Friday
    ]


def _nth_weekday(year: int, month: int, weekday: int, n: int) -> date:
    """Return the nth occurrence of a weekday in a month.

    weekday: 0=Monday, 1=Tuesday, ..., 6=Sunday
    n: 1-based occurrence (1=first, 4=fourth)
    """
    from calendar import monthrange
    first_day, days_in_month = monthrange(year, month)
    day = 1 + (weekday - first_day) % 7 + (n - 1) * 7
    if day > days_in_month:
        day -= 7
    return date(year, month, day)


def _last_monday(year: int, month: int) -> date:
    """Return the last Monday of a given month."""
    from calendar import monthrange
    _, days_in_month = monthrange(year, month)
    last_day = date(year, month, days_in_month)
    offset = (last_day.weekday() - 0) % 7
    return date(year, month, days_in_month - offset)


def _juneteenth(year: int) -> date:
    """Juneteenth National Independence Day (observed)."""
    d = date(year, 6, 19)
    if d.weekday() == 5:  # Saturday -> Friday
        return date(year, 6, 18)
    if d.weekday() == 6:  # Sunday -> Monday
        return date(year, 6, 20)
    return d


def _good_friday(year: int) -> date:
    """Approximate Good Friday using the Computus (Gauss algorithm)."""
    a = year % 19
    b = year // 100
    c = year % 100
    d = b // 4
    e = b % 4
    f = (b + 8) // 25
    g = (b - f + 1) // 3
    h = (19 * a + b - d - g + 15) % 30
    i = c // 4
    k = c % 4
    l = (32 + 2 * e + 2 * i - h - k) % 7
    m = (a + 11 * h + 22 * l) // 451
    month = (h + l - 7 * m + 114) // 31
    day = ((h + l - 7 * m + 114) % 31) + 1
    easter = date(year, month, day)
    return easter - date.resolution * 2  # Good Friday = Easter Sunday - 2 days


_NYSE_HOLIDAYS: dict[int, set[date]] = {}


def _get_holidays(year: int) -> set[date]:
    """Return cached set of NYSE holidays for a year."""
    if year not in _NYSE_HOLIDAYS:
        holidays = set(_year_holidays(year))
        observed = set()
        for h in holidays:
            w = h.weekday()
            if w == 5:  # Saturday -> observed Friday
                observed.add(date(h.year, h.month, h.day - 1))
            elif w == 6:  # Sunday -> observed Monday
                observed.add(date(h.year, h.month, h.day + 1))
        _NYSE_HOLIDAYS[year] = holidays | observed
    return _NYSE_HOLIDAYS[year]


def is_nyse_holiday(d: date) -> bool:
    """Check if a date is a NYSE holiday (market closed)."""
    return d in _get_holidays(d.year)


def is_trading_day(d: date) -> bool:
    """Check if a date is a US market trading day (Mon-Fri, not a holiday)."""
    if d.weekday() >= 5:
        return False
    return not is_nyse_holiday(d)


def get_trading_days(start: date, end: date) -> list[date]:
    """Return list of US market trading days in a date range.

    Args:
        start: Start date (inclusive).
        end: End date (inclusive).

    Returns:
        List of trading days (Mon-Fri, excluding NYSE holidays).
    """
    import datetime as _datetime
    days = []
    current = start
    while current <= end:
        if is_trading_day(current):
            days.append(current)
        current += _datetime.timedelta(days=1)
    return days
