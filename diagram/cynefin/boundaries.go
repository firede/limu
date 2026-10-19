package cynefin

import (
	"math"
	"unicode/utf16"
)

// The geometry of Mermaid's Cynefin boundaries, as Pele's boundaries.ts has it: seeded wavy
// lines between the domains and the S-shaped cliff between Clear and Chaotic. Curves are flat
// lists of numbers: a start point, then six numbers (two control points and an end point) per
// cubic segment.
//
// The seeded numbers are worked out with JavaScript's conversions between doubles and 32-bit
// integers, which wrap: toInt32 is what `| 0` and `^` make of a double, imul is Math.imul,
// and >>> shifts the integer's bits as an unsigned one.

// toInt32 is JavaScript's ToInt32: the whole part of x, wrapped to 32 bits, and 0 for NaN and
// the infinities.
func toInt32(x float64) int32 {
	if math.IsNaN(x) || math.IsInf(x, 0) {
		return 0
	}
	return int32(uint32(int64(math.Mod(math.Trunc(x), 1<<32))))
}

// imul is Math.imul: the low 32 bits of the product of two 32-bit integers.
func imul(a, b int32) int32 { return int32(uint32(a) * uint32(b)) }

// ushr is a >>> n, as a signed integer again, which is what ^ makes of it.
func ushr(a int32, n uint) int32 { return int32(uint32(a) >> n) }

// seededRandom is one step of mulberry32 from seed, a number in [0, 1), as Pele's
// seededRandom works it out.
func seededRandom(seed float64) float64 {
	t := toInt32(seed + 0x6d2b79f5)
	t = imul(t^ushr(t, 15), t|1)
	t ^= t + imul(t^ushr(t, 7), t|61)
	return float64(uint32(t^ushr(t, 14))) / 4294967296
}

// hashString is Pele's hashString: Java's hash of a string's UTF-16 code units, in 32 bits.
func hashString(s string) int32 {
	var hash int32
	for _, c := range utf16.Encode([]rune(s)) {
		hash = hash<<5 - hash + int32(c)
	}
	return hash
}

// resolveSeed is Pele's resolveSeed: a configured seed other than zero wins; otherwise the
// seed is the hash of the diagram's id.
func resolveSeed(configured float64, id string) float64 {
	if !math.IsInf(configured, 0) && !math.IsNaN(configured) && configured != 0 {
		return configured
	}
	return float64(hashString(id))
}

// waveOptions are Pele's WaveOptions: the segments of a wavy line, whether it runs down, the
// multipliers Mermaid varies the seed by per point and per control point, and whether its
// ends stay on its center line.
type waveOptions struct {
	segments                             int
	vertical                             bool
	pointStep, controlStep, controlShift float64
	pinStart, pinEnd                     bool
}

// fold and horizon are Pele's FOLD and HORIZON.
var (
	fold    = waveOptions{segments: 7, vertical: true, pointStep: 17, controlStep: 31, controlShift: 7}
	horizon = waveOptions{segments: 7, vertical: false, pointStep: 23, controlStep: 37, controlShift: 11}
)

// wave is a wavy line along one axis from `from` to `to`, centered on center in the other
// axis, as Pele's wave draws it: points jitter by up to the amplitude, and a pinned end stays
// on the center line.
func wave(from, to, center, seed, amplitude float64, o waveOptions) []float64 {
	step := (to - from) / float64(o.segments)
	along := make([]float64, 0, o.segments+1)
	across := make([]float64, 0, o.segments+1)
	for i := 0; i <= o.segments; i++ {
		fi := float64(i)
		pinned := i == 0 && o.pinStart || i == o.segments && o.pinEnd
		along = append(along, from+float64(fi*step))
		if pinned {
			across = append(across, center)
		} else {
			r := seededRandom(seed + float64(fi*o.pointStep))
			across = append(across, center+float64(r*amplitude*2)-amplitude)
		}
	}
	out := []float64{along[0], across[0]}
	if o.vertical {
		out = []float64{across[0], along[0]}
	}
	for i := 0; i < o.segments; i++ {
		mid := (along[i] + along[i+1]) / 2
		sign := 1.0
		if i%2 != 0 {
			sign = -1
		}
		r := seededRandom(seed + float64(float64(i)*o.controlStep) + o.controlShift)
		offset := float64(amplitude * 1.5 * sign * r)
		if o.vertical {
			out = append(out, across[i]+offset, mid, across[i+1]-offset, mid, across[i+1], along[i+1])
		} else {
			out = append(out, mid, across[i]+offset, mid, across[i+1]-offset, along[i+1], across[i+1])
		}
	}
	return out
}

// cliff is the steep S from (cx, top) down to (cx, bottom), as Pele's cliff draws it.
func cliff(cx, top, bottom, amplitude float64) []float64 {
	h := bottom - top
	return []float64{
		cx, top,
		cx + amplitude, top + float64(h*0.2), cx - float64(amplitude*1.5), top + float64(h*0.55), cx + float64(amplitude*0.5), top + float64(h*0.75),
		cx - amplitude, top + float64(h*0.85), cx + float64(amplitude*0.3), top + float64(h*0.95), cx, bottom,
	}
}
