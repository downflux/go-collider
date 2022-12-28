package kinematics

import (
	"time"

	"github.com/downflux/go-collider/agent"
	"github.com/downflux/go-collider/feature"
	"github.com/downflux/go-geometry/2d/vector"
	"github.com/downflux/go-geometry/2d/vector/polar"
	"github.com/downflux/go-geometry/epsilon"

	chr "github.com/downflux/go-collider/internal/geometry/hyperrectangle"
)

// SetCollisionVelocityStrict geenerates a velocity vector for two colliding
// objects.
//
// The input velocity vector v is a velocity buffer for the first agent a; if
// this vector points towards the neighbor b, then the vector as a whole is set
// to zero -- that is, a is forced to stop for the current tick.
//
// This is a much simpler way to deal with the three body problem -- this, the
// case of when the constant "flip-flip" from SetCollisionVelocity can
// accidentally flip the velocity vector back into a neighbor.
func SetCollisionVelocityStrict(a *agent.A, b *agent.A, v vector.M) {
	// Find the unit collision vector pointing from a to b.
	buf := vector.M{0, 0}
	buf.Copy(b.Position())
	buf.Sub(a.Position())

	// If the vectors are pointing in the same direction, then force the
	// object to stop moving.
	if c := vector.Dot(buf.V(), v.V()); c > 0 {
		v.SetX(0)
		v.SetY(0)
	}
}

func SetFeatureCollisionVelocityStrict(a *agent.A, f *feature.F, v vector.M) {
	n := chr.N(f.AABB(), a.Position()).M()
	n.Scale(-1)
	if c := vector.Dot(n.V(), v.V()); c > 0 {
		v.SetX(0)
		v.SetY(0)
	}
}

// SetCollisionVelocity generates a velocity vector for two colliding objects by
// setting the normal components to zero. This does not model, and does not
// intend to model, an inelastic collision -- this is the final check we do to
// avoid odd rendering behavior where a unit is forced into a collision.
//
// In the case of an inelastic collision between three objects, we can imagine a
// small object stuck between two massive ones. The collision vector for either
// massive object is not changed very much by the middle object, but because the
// two massive objects are not yet colliding, they will continue moving towards
// one another, which will force a collision with the middle object.
//
// The collision avoidance layer should generate velocities for each agent which
// anticipates collisions (which itself may be modeled as a near-field inelastic
// repulsive force) to avoid unintuitive pathing behavior.
//
// This function does not check if we need to generate a collision velocity in
// the first place -- that should be done by the caller by e.g. checking for
// radius overlap.
//
// To generate a final collision vector between several colliding objects, this
// function should be called iteratively for a single object and other
// colliders, e.g.
//
//	v := vector.M{0, 0}
//	v.Copy(a.Velocity())
//
//	SetCollisionVelocity(a, b, v)
//	SetCollisionVelocity(a, c, v)
//
// This allows us to generate a final velocity for agent a.
//
// N.B.: This may generate a velocity vector which flips back into a forbidden
// zone. In order to take this into account, the caller must do two passes,
// where the second pass calls SetCollisionVelocityStrict to force the velocity
// to zero in case of continued velocity violations.
func SetCollisionVelocity(a *agent.A, b *agent.A, v vector.M) {
	// Find the unit collision vector pointing from a to b.
	buf := vector.M{0, 0}
	buf.Copy(b.Position())
	buf.Sub(a.Position())
	buf.Unit()

	// If the vectors are pointing in the same direction, then force the
	// object to stop moving.
	if c := vector.Dot(buf.V(), v.V()); c > 0 {
		buf.Scale(c)
		v.Sub(buf.V())
	}
}

func SetFeatureCollisionVelocity(a *agent.A, f *feature.F, v vector.M) {
	n := chr.N(f.AABB(), a.Position()).M()
	n.Scale(-1)
	if c := vector.Dot(n.V(), v.V()); c > 0 {
		n.Scale(c)
		v.Sub(n.V())
	}
}

// SetVelocity clamps the agent velocity to the maximum possible magnitude
// defind by the max velocity and max acceleration.
func SetVelocity(a *agent.A, v vector.M) {
	if c := vector.Magnitude(v.V()); c > a.MaxVelocity() {
		v.Scale(a.MaxVelocity() / c)
	}

	buf := vector.M{0, 0}
	buf.Copy(v.V())
	buf.Sub(a.Velocity())

	// Do not apply acceleration limits for breaking.
	if vector.Magnitude(a.Velocity()) < vector.Magnitude(v.V()) {
		if c := vector.Magnitude(buf.V()); c > a.MaxAcceleration() {
			buf.Scale(a.MaxAcceleration() / c)
			v.Copy(a.Velocity())
			v.Add(buf.V())
		}
	}
}

// SetHeading sets the input velocity and heading vectors to the appropriate
// simulated values for the next tick.
//
// TODO(minkezhang): Handle agents that can reverse.
func SetHeading(a *agent.A, d time.Duration, v vector.M, h polar.M) {
	if epsilon.Within(vector.Magnitude(v.V()), 0) {
		return
	}

	h.Copy(a.Heading())
	p := polar.Polar(v.V())

	// We do not need to worry about scaling v by t, as we only care about
	// the angular difference between v and the heading.
	omega := a.MaxAngularVelocity() * (float64(d) / float64(time.Second))

	if a.Heading().Theta() > p.Theta() {
		if a.Heading().Theta()-p.Theta() > omega {
			h.SetTheta(a.Heading().Theta() - omega)
			v.SetX(0)
			v.SetY(0)
		} else {
			h.SetTheta(p.Theta())
		}
	} else if a.Heading().Theta() < p.Theta() {
		if p.Theta()-a.Heading().Theta() > omega {
			h.SetTheta(a.Heading().Theta() + omega)
			v.SetX(0)
			v.SetY(0)
		} else {
			h.SetTheta(p.Theta())
		}
	}

	h.Normalize()
}