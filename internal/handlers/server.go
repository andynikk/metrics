package handlers

type MetricError int

func (et MetricError) String() string {
	return [...]string{"Not error", "Error convert", "Error get type"}[et]
}
