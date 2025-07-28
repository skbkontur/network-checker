package quantile

type Result struct {
	Count int
	P50   float64
	P90   float64
	P99   float64
}

func (s *Stream) Result() *Result {
	return &Result{
		Count: s.Count(),
		P50:   s.Query(0.50),
		P90:   s.Query(0.90),
		P99:   s.Query(0.99),
	}
}
