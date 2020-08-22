package writer

//
// Type is an enum that represents a type of data point to be written out.
//
type Type int

const (
  ClosingPrice Type = iota
  GrossMockEarnings
)

func (o Type) String() string {
  return [...]string{"ClosingPrice", "GrossMockEarnings"}[o]
}