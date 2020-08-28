package constants

import (
  "github.com/shopspring/decimal"
  "time"
)

const (
  LogPrefixFmt = "%-17s "
  TwelveHours = 12 * time.Hour
)

var (
  negOne = decimal.NewFromInt(-1)
  one    = decimal.NewFromInt(1)
  two    = decimal.NewFromInt(2)
)

func NegOne() decimal.Decimal {
  return negOne
}

func One() decimal.Decimal {
  return one
}

func Two() decimal.Decimal {
  return two
}