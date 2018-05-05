package sampler

// A Sampler samplers from the given model. Note that we follow gonum
// conventions and sample in place (although we return an error). Although
// it's not explicit, it is assumed that the sample is the same size as a
// variable list
type Sampler interface {
	Sample(s []float64) error
}
