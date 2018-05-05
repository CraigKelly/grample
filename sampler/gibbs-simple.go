package sampler

import (
	"github.com/CraigKelly/grample/model"
	"github.com/pkg/errors"
)

// GibbsSimple is our baseline, simple to code Gibbs sampler
type GibbsSimple struct {
	pgm  *model.Model
	last []*model.Variable
}

// NewGibbsSimple creates a new sampler
func NewGibbsSimple(m *model.Model) (*GibbsSimple, error) {
	if m == nil {
		return nil, errors.New("No model supplied")
	}

	s := &GibbsSimple{
		pgm: m,
	}
	return s, nil
}

// Sample returns a single sample
func (g *GibbsSimple) Sample() ([]*model.Variable, error) {
	if len(g.last) < 1 {
		// Initial: just initialize randomly
		last := make([]*model.Variable, len(g.pgm.Vars))
		return last, nil
	}
	return nil, nil //TODO
}
