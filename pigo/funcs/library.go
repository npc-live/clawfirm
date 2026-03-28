package funcs

import (
	"fmt"
	"log"
	"math"
	"time"

	"github.com/elliotchance/pie/pie"
	"gonum.org/v1/gonum/stat"
)

const float64EqualityThreshold = 1e-5

// defer timeTrack(time.Now(), "factorial")

func almostEqual(a, b float64) bool {
	return math.Abs(a-b) <= float64EqualityThreshold
}

//RollingIterator is for
type RollingIterator interface {
	Iterator(window int) <-chan []float64
}

type floatRollingIterator []float64

func (r floatRollingIterator) Iterator(length int) <-chan []float64 {
	c := make(chan []float64)
	lenR := len(r)
	go func() {
		defer close(c)
		for i := lenR - 1; i >= 0; i-- {
			size := lenR - i
			if size < length {
				c <- nil
			} else {
				c <- r[i:][:length]
			}
		}
	}()
	return c
}

//Abs is for
func Abs(params ...interface{}) interface{} {
	if len(params) != 1 {
		return fmt.Errorf("params not matched")
	}
	p := params[0]

	switch p.(type) {
	case pie.Float64s:
		return p.(pie.Float64s).Abs()
	}

	return fmt.Errorf("types not matched")
}

//Sum is for
func Sum(params ...interface{}) interface{} {
	if len(params) != 2 {
		return fmt.Errorf("params not matched")
	}
	p := params[0]
	p1 := params[1]
	switch p.(type) {
	case pie.Float64s:
		switch p1.(type) {
		case float64:
			ret := pie.Float64s{}
			for r := range floatRollingIterator(p.(pie.Float64s)).Iterator(int(p1.(float64))) {
				ret = append([]float64{pie.Float64s(r).Sum()}, ret...)
			}
			return ret
		}
	}

	return fmt.Errorf("types not matched")
}

//Max is for
func Max(params ...interface{}) interface{} {
	if len(params) != 2 {
		return fmt.Errorf("params not matched")
	}
	p := params[0]
	p1 := params[1]
	switch p.(type) {
	case pie.Float64s:
		switch p1.(type) {
		case pie.Float64s:
			ret := pie.Float64s{}
			p1pie := p.(pie.Float64s)
			p2pie := p1.(pie.Float64s)
			for i := 0; i < p1pie.Len(); i++ {
				ret = append(ret, math.Max(p2pie[i], p1pie[i]))
			}
			return ret
		}
	}
	return fmt.Errorf("types not matched")
}

//Min is for
func Min(params ...interface{}) interface{} {
	if len(params) != 2 {
		return fmt.Errorf("params not matched")
	}
	p := params[0]
	p1 := params[1]
	switch p.(type) {
	case pie.Float64s:
		switch p1.(type) {
		case pie.Float64s:
			ret := pie.Float64s{}
			p1pie := p.(pie.Float64s)
			p2pie := p1.(pie.Float64s)
			for i := p1pie.Len() - 1; i >= 1; i-- {
				ret = pappend(ret, math.Min(p2pie[i], p1pie[i]))
			}
			ret = pappend(ret, 0.0)
			return ret
		}
	}
	return fmt.Errorf("types not matched")
}

//ACos is for
func ACos(params ...interface{}) interface{} {
	if len(params) != 1 {
		return fmt.Errorf("params not matched")
	}
	p := params[0]
	switch p.(type) {
	case pie.Float64s:
		ret := pie.Float64s{}
		p1pie := p.(pie.Float64s)
		for i := p1pie.Len() - 1; i >= 1; i-- {
			ret = pappend(ret, math.Acos(p1pie[i]))
		}
		ret = pappend(ret, 0.0)
		return ret
	}
	return fmt.Errorf("types not matched")
}

//ASin is for
func ASin(params ...interface{}) interface{} {
	if len(params) != 1 {
		return fmt.Errorf("params not matched")
	}
	p := params[0]
	switch p.(type) {
	case pie.Float64s:
		ret := pie.Float64s{}
		p1pie := p.(pie.Float64s)
		for i := p1pie.Len() - 1; i >= 1; i-- {
			ret = pappend(ret, math.Asin(p1pie[i]))
		}
		ret = pappend(ret, 0.0)
		return ret
	}
	return fmt.Errorf("types not matched")
}

//ATan is for
func ATan(params ...interface{}) interface{} {
	if len(params) != 1 {
		return fmt.Errorf("params not matched")
	}
	p := params[0]
	switch p.(type) {
	case pie.Float64s:
		ret := pie.Float64s{}
		p1pie := p.(pie.Float64s)
		for i := p1pie.Len() - 1; i >= 1; i-- {
			ret = pappend(ret, math.Atan(p1pie[i]))
		}
		ret = pappend(ret, 0.0)
		return ret
	}
	return fmt.Errorf("types not matched")
}

//Cos is for
func Cos(params ...interface{}) interface{} {
	if len(params) != 1 {
		return fmt.Errorf("params not matched")
	}
	p := params[0]
	switch p.(type) {
	case pie.Float64s:
		ret := pie.Float64s{}
		p1pie := p.(pie.Float64s)
		for i := p1pie.Len() - 1; i >= 1; i-- {
			ret = pappend(ret, math.Cos(p1pie[i]))
		}
		ret = pappend(ret, 0.0)
		return ret
	}
	return fmt.Errorf("types not matched")
}

//Sin is for
func Sin(params ...interface{}) interface{} {
	if len(params) != 1 {
		return fmt.Errorf("params not matched")
	}
	p := params[0]
	switch p.(type) {
	case pie.Float64s:
		ret := pie.Float64s{}
		p1pie := p.(pie.Float64s)
		for i := p1pie.Len() - 1; i >= 1; i-- {
			ret = pappend(ret, math.Sin(p1pie[i]))
		}
		ret = pappend(ret, 0.0)
		return ret
	}
	return fmt.Errorf("types not matched")
}

//Tan is for
func Tan(params ...interface{}) interface{} {
	if len(params) != 1 {
		return fmt.Errorf("params not matched")
	}
	p := params[0]
	switch p.(type) {
	case pie.Float64s:
		ret := pie.Float64s{}
		p1pie := p.(pie.Float64s)
		for i := p1pie.Len() - 1; i >= 1; i-- {
			ret = pappend(ret, math.Tan(p1pie[i]))
		}
		ret = pappend(ret, 0.0)
		return ret
	}
	return fmt.Errorf("types not matched")
}

//Exp is for
func Exp(params ...interface{}) interface{} {
	if len(params) != 1 {
		return fmt.Errorf("params not matched")
	}
	p := params[0]
	switch p.(type) {
	case pie.Float64s:
		ret := pie.Float64s{}
		p1pie := p.(pie.Float64s)
		for i := p1pie.Len() - 1; i >= 1; i-- {
			ret = pappend(ret, math.Exp(p1pie[i]))
		}
		ret = pappend(ret, 0.0)
		return ret
	}
	return fmt.Errorf("types not matched")
}

//Log is for
func Log(params ...interface{}) interface{} {
	if len(params) != 1 {
		return fmt.Errorf("params not matched")
	}
	p := params[0]
	switch p.(type) {
	case pie.Float64s:
		ret := pie.Float64s{}
		p1pie := p.(pie.Float64s)
		for i := p1pie.Len() - 1; i >= 1; i-- {
			ret = pappend(ret, math.Log(p1pie[i]))
		}
		ret = pappend(ret, 0.0)
		return ret
	}
	return fmt.Errorf("types not matched")
}

//Ln is for
func Ln(params ...interface{}) interface{} {
	if len(params) != 1 {
		return fmt.Errorf("params not matched")
	}
	p := params[0]
	switch p.(type) {
	case pie.Float64s:
		ret := pie.Float64s{}
		p1pie := p.(pie.Float64s)
		for i := p1pie.Len() - 1; i >= 1; i-- {
			ret = pappend(ret, math.Log10(p1pie[i]))
		}
		ret = pappend(ret, 0.0)
		return ret
	}
	return fmt.Errorf("types not matched")
}

//Sqrt is for
func Sqrt(params ...interface{}) interface{} {
	if len(params) != 1 {
		return fmt.Errorf("params not matched")
	}
	p := params[0]
	switch p.(type) {
	case pie.Float64s:
		ret := pie.Float64s{}
		p1pie := p.(pie.Float64s)
		for i := p1pie.Len() - 1; i >= 1; i-- {
			ret = pappend(ret, math.Sqrt(p1pie[i]))
		}
		ret = pappend(ret, 0.0)
		return ret
	}
	return fmt.Errorf("types not matched")
}

//Ceiling is for
func Ceiling(params ...interface{}) interface{} {
	if len(params) != 1 {
		return fmt.Errorf("params not matched")
	}
	p := params[0]
	switch p.(type) {
	case pie.Float64s:
		ret := pie.Float64s{}
		p1pie := p.(pie.Float64s)
		for i := p1pie.Len() - 1; i >= 1; i-- {
			ret = pappend(ret, math.Ceil(p1pie[i]))
		}
		ret = pappend(ret, 0.0)
		return ret
	}
	return fmt.Errorf("types not matched")
}

//Floor is for
func Floor(params ...interface{}) interface{} {
	if len(params) != 1 {
		return fmt.Errorf("params not matched")
	}
	p := params[0]
	switch p.(type) {
	case pie.Float64s:
		ret := pie.Float64s{}
		p1pie := p.(pie.Float64s)
		for i := p1pie.Len() - 1; i >= 1; i-- {
			ret = pappend(ret, math.Floor(p1pie[i]))
		}
		ret = pappend(ret, 0.0)
		return ret
	}
	return fmt.Errorf("types not matched")
}

//Intpart is for
func Intpart(params ...interface{}) interface{} {
	if len(params) != 1 {
		return fmt.Errorf("params not matched")
	}
	p := params[0]
	switch p.(type) {
	case pie.Float64s:
		ret := pie.Float64s{}
		p1pie := p.(pie.Float64s)
		for i := p1pie.Len() - 1; i >= 1; i-- {
			intpart, _ := math.Modf(p1pie[i])
			ret = pappend(ret, float64(intpart))
		}
		ret = pappend(ret, 0.0)
		return ret
	}
	return fmt.Errorf("types not matched")
}

//FracPart is for
func FracPart(params ...interface{}) interface{} {
	if len(params) != 1 {
		return fmt.Errorf("params not matched")
	}
	p := params[0]
	switch p.(type) {
	case pie.Float64s:
		ret := pie.Float64s{}
		p1pie := p.(pie.Float64s)
		for i := p1pie.Len() - 1; i >= 1; i-- {
			_, f := math.Modf(p1pie[i])
			ret = pappend(ret, f)
		}
		ret = pappend(ret, 0.0)
		return ret
	}
	return fmt.Errorf("types not matched")
}

//Round is for
func Round(params ...interface{}) interface{} {
	if len(params) != 1 {
		return fmt.Errorf("params not matched")
	}
	p := params[0]
	switch p.(type) {
	case pie.Float64s:
		ret := pie.Float64s{}
		p1pie := p.(pie.Float64s)
		for i := p1pie.Len() - 1; i >= 1; i-- {
			ret = pappend(ret, math.Round(p1pie[i]))
		}
		ret = pappend(ret, 0.0)
		return ret
	}
	return fmt.Errorf("types not matched")
}

func Corr(params ...interface{}) interface{} {
	//loging.Info("Corr", "params", params)
	if len(params) != 3 {
		return fmt.Errorf("params not matched")
	}

	p0 := params[0]
	p1 := params[1]
	p2f := params[2]
	//ln("Slope", p1, p)
	switch p0.(type) {
	case pie.Float64s:
		switch p2f.(type) {
		case float64:
			ret := pie.Float64s{}
			n := int(p2f.(float64))
			iter := floatRollingIterator(p1.(pie.Float64s)).Iterator(n)
			for r := range floatRollingIterator(p0.(pie.Float64s)).Iterator(n) {
				r2 := <-iter
				if r != nil {
					corr := math.Abs(stat.Correlation(r2, r, nil))
					ret = append([]float64{corr}, ret...)
				} else {
					ret = append([]float64{0.0}, ret...)
				}
			}

			return ret
		}
	}

	return fmt.Errorf("types not matched")
}

//Slope is for
func Slope(params ...interface{}) interface{} {
	if len(params) != 2 {
		return fmt.Errorf("params not matched")
	}
	p := params[0]
	p1 := params[1]
	//ln("Slope", p1, p)
	switch p.(type) {
	case pie.Float64s:
		switch p1.(type) {
		case float64:
			ret := pie.Float64s{}
			n := int(p1.(float64))
			mySlice := make([]float64, int(p1.(float64)))
			for i := 0; i < n; i++ {
				mySlice[i] = float64(i)
			}
			for r := range floatRollingIterator(p.(pie.Float64s)).Iterator(n) {
				if r != nil {
					a, _ := LeastSquares(mySlice, r)
					ret = append([]float64{a}, ret...)

				} else {
					ret = append([]float64{0.0}, ret...)
				}
			}

			return ret
		}
	}

	return fmt.Errorf("types not matched")
}

// LeastSquares is for
func LeastSquares(x []float64, y []float64) (a float64, b float64) {
	// x是横坐标数据,y是纵坐标数据
	// a是斜率，b是截距
	// fmt.Println(x, y)
	xi := float64(0)
	x2 := float64(0)
	yi := float64(0)
	xy := float64(0)
	length := float64(len(x))
	for i := 0; i < len(x); i++ {
		xi += x[i]
		x2 += x[i] * x[i]
		yi += y[i]
		xy += x[i] * y[i]
	}
	a = (yi*xi - xy*length) / (xi*xi - x2*length)
	b = (yi*x2 - xy*xi) / (x2*length - xi*xi)
	return
}

//Sign is for
func Sign(params ...interface{}) interface{} {
	if len(params) != 1 {
		return fmt.Errorf("params not matched")
	}
	p := params[0]
	switch p.(type) {
	case pie.Float64s:
		ret := pie.Float64s{}
		p1pie := p.(pie.Float64s)
		for i := p1pie.Len() - 1; i >= 1; i-- {
			if p1pie[i] > 0.0 {
				ret = pappend(ret, p1pie[i])
			} else {
				ret = pappend(ret, p1pie[i])
			}
		}
		ret = pappend(ret, 0.0)
		return ret
	}
	return fmt.Errorf("types not matched")
}

//Mod is for
func Mod(params ...interface{}) interface{} {
	if len(params) != 1 {
		return fmt.Errorf("params not matched")
	}
	p := params[0]
	p1 := params[1]
	switch p.(type) {
	case pie.Float64s:
		switch p1.(type) {
		case float64:
			ret := pie.Float64s{}
			m := p1.(float64)
			p1pie := p.(pie.Float64s)
			for i := p1pie.Len() - 1; i >= 1; i-- {
				ret = pappend(ret, math.Mod(p1pie[i], m))
			}
			ret = pappend(ret, 0.0)
			return ret
		}
	}
	return fmt.Errorf("types not matched")
}

//Stddev is for
func Stddev(params ...interface{}) interface{} {
	if len(params) != 2 {
		return fmt.Errorf("params not matched")
	}
	p := params[0]
	p1 := params[1]
	switch p.(type) {
	case pie.Float64s:
		switch p1.(type) {
		case float64:
			ret := pie.Float64s{}
			for r := range floatRollingIterator(p.(pie.Float64s)).Iterator(int(p1.(float64))) {
				ret = append([]float64{pie.Float64s(r).Stddev()}, ret...)
			}
			return ret
		}
	}

	return fmt.Errorf("types not matched")
}
func pappend(list []float64, r float64) []float64 {
	return append([]float64{r}, list...)
}

//Sma is for
func Sma(params ...interface{}) interface{} {
	if len(params) != 3 {
		return fmt.Errorf("params not matched")
	}
	p := params[0]
	p1 := params[1]
	p2 := params[2]
	var p1pie pie.Float64s
	var p2f float64
	var p3f float64
	var ok bool = false
	if p1pie, ok = p.(pie.Float64s); ok == false {
		return fmt.Errorf("types not matched")
	}
	if p2f, ok = p1.(float64); ok == false {
		return fmt.Errorf("types not matched")
	}
	if p3f, ok = p2.(float64); ok == false {
		return fmt.Errorf("types not matched")
	}
	ret := pie.Float64s{}
	a := p3f / p2f
	prev := 0.0
	length := int(p3f)
	//lenR := p1pie.Len()
	count := 0
	for i := p1pie.Len() - 1; i >= 0; i-- {
		if count < length {
			ret = pappend(ret, 0.0)
		} else if count == length {
			prev = p1pie[i:].Average()
			ret = pappend(ret, prev)
		} else {
			ema := a*p1pie[i] + (1-a)*prev
			ret = pappend(ret, ema)
			prev = ema
		}
		count++

	}
	return ret

}

//Ema is for
func Ema(params ...interface{}) interface{} {
	if len(params) != 2 {
		return fmt.Errorf("params not matched")
	}
	p := params[0]
	p1 := params[1]
	switch p.(type) {
	case pie.Float64s:
		switch p1.(type) {
		case float64:
			p0pie := p.(pie.Float64s)
			p1f := p1.(float64)
			length := int(p1f)
			ret := pie.Float64s{}
			a := 2 / (p1f + 1)
			prev := 0.0
			lenR := p0pie.Len()
			for i := p0pie.Len() - 1; i >= 0; i-- {
				size := lenR - i
				if size < length {
					ret = pappend(ret, p0pie[i])
					prev = ret.Average()
					//fmt.Println("1", prev, ret)
				} else {
					ema := a*p0pie[i] + (1-a)*prev
					ret = pappend(ret, ema)
					prev = ema
				}
			}
			return ret
		}
	}

	return fmt.Errorf("types not matched")
}

//Ma is for
func Ma(params ...interface{}) interface{} {
	if len(params) != 2 {
		return fmt.Errorf("params not matched")
	}
	p := params[0]
	p1 := params[1]
	//fmt.Println("p,p1", p.(pie.Float64s), p1)
	switch p.(type) {
	case pie.Float64s:
		switch p1.(type) {
		case float64:
			ret := pie.Float64s{}
			data := p.(pie.Float64s)
			windowSize := int(p1.(float64))

			// For time-descending data (newest first)
			// MA(data,7) at index i should use data[i] to data[i+6] (if available)
			for i := 0; i < len(data); i++ {
				// Calculate how many data points we can use for the window
				endIdx := i + windowSize
				if endIdx > len(data) {
					endIdx = len(data)
				}

				// Calculate average of available data points from current position onwards
				windowData := data[i:endIdx]
				if len(windowData) > 0 {
					ret = append(ret, windowData.Average())
				} else {
					ret = append(ret, 0.0)
				}
			}
			return ret
		}
	}

	return fmt.Errorf("types not matched")
}

//Cross is for
func Cross(params ...interface{}) interface{} {
	if len(params) != 2 {
		return fmt.Errorf("params not matched")
	}
	p := params[0]
	p1 := params[1]
	switch p.(type) {
	case pie.Float64s:
		p1pie := p.(pie.Float64s)
		switch p1.(type) {
		case pie.Float64s:
			p2pie := p1.(pie.Float64s)
			ret := pie.Float64s{}
			//for i := 1; i < p1pie.Len(); i++ {
			for i := p1pie.Len() - 1; i >= 1; i-- {
				c := 0.0
				//loging.Info("Cross", "index", i, "value", []float64{p1pie[i], p2pie[i], p1pie[i-1], p2pie[i-1]})
				if p1pie[i] > p2pie[i] && p1pie[i-1] < p2pie[i-1] {
					c = -1.0
				}
				if p1pie[i] < p2pie[i] && p1pie[i-1] > p2pie[i-1] {
					c = 1.0
				}
				ret = pappend(ret, c)
			}
			ret = append(ret, 0.0)
			return ret
		}
	}

	return fmt.Errorf("types not matched")
}

//LongCross is for
func LongCross(params ...interface{}) interface{} {
	if len(params) != 3 {
		return fmt.Errorf("params not matched")
	}
	p := params[0]
	p1 := params[1]
	length := 0.0
	ok := false
	if length, ok = params[2].(float64); ok == false {
		return fmt.Errorf("params not matched")
	}

	switch p.(type) {
	case pie.Float64s:
		p1pie := p.(pie.Float64s)
		switch p1.(type) {
		case pie.Float64s:
			p2pie := p1.(pie.Float64s)
			ret := pie.Float64s{}
			n := int(length)

			for i := n - 1; i >= 0; i-- {
				ret = pappend(ret, 0.0)
			}

			for i := p1pie.Len() - n - 1; i >= 0; i-- {
				c := 0
				startPos := i + n // n = 1
				if p1pie[i] > p2pie[i] && p1pie[startPos] < p2pie[startPos] {
					c = -1
				}
				if p1pie[i] < p2pie[i] && p1pie[startPos] > p2pie[startPos] {
					c = 1
				}
				for prevIdx := i + 1; prevIdx < startPos; prevIdx++ {
					if c == 1 && p1pie[prevIdx] < p2pie[prevIdx] {
						c = 0
					} else if c == -1 && p1pie[prevIdx] > p2pie[prevIdx] {
						c = 0
					} else {

					}
				}
				ret = pappend(ret, float64(c))
			}
			return ret
		}
	}

	return fmt.Errorf("types not matched")
}

//UpNDay is for
func UpNDay(params ...interface{}) interface{} {
	if len(params) != 2 {
		return fmt.Errorf("params not matched")
	}
	p := params[0]
	//p1 := params[1]
	length := 0.0
	ok := false
	if length, ok = params[1].(float64); ok == false {
		return fmt.Errorf("params not matched")
	}
	switch p.(type) {
	case pie.Float64s:
		p1pie := p.(pie.Float64s)
		ret := pie.Float64s{}
		n := int(length)
		for i := n - 1; i > 0; i-- {
			ret = append(ret, 0.0)
		}
		for i := p1pie.Len() - n; i >= 0; i-- {
			iPos := i
			c := 0
			startPos := iPos + n // n = 1
			prev := 0.0
			if p1pie[iPos] > p1pie[i+1] {
				c = 1
				prev = p1pie[iPos]
			}
			for prevIdx := i + 1; prevIdx < startPos; prevIdx++ {
				if c == 1 && prev > p1pie[prevIdx] {
					prev = p1pie[prevIdx]
				} else {
					c = 0
				}
			}
			ret = pappend(ret, float64(c))
		}
		return ret

	}

	return fmt.Errorf("types not matched")
}

//DownNDay is for
func DownNDay(params ...interface{}) interface{} {
	if len(params) != 2 {
		return fmt.Errorf("params not matched")
	}
	p := params[0]
	//p1 := params[1]
	length := 0.0
	ok := false
	if length, ok = params[1].(float64); ok == false {
		return fmt.Errorf("params not matched")
	}
	switch p.(type) {
	case pie.Float64s:
		p1pie := p.(pie.Float64s)
		ret := pie.Float64s{}
		n := int(length)
		for i := n - 1; i > 0; i-- {
			ret = append(ret, 0.0)
		}
		for i := p1pie.Len() - n; i >= 0; i-- {
			iPos := i
			c := 0
			startPos := iPos + n // n = 1
			prev := 0.0
			if p1pie[iPos] < p1pie[i+1] {
				c = 1
				prev = p1pie[iPos]
			}
			for prevIdx := i + 1; prevIdx < startPos; prevIdx++ {

				if c == 1 && prev < p1pie[prevIdx] {
					prev = p1pie[prevIdx]
				} else {
					c = 0
				}
			}
			ret = pappend(ret, float64(c))
		}
		return ret

	}

	return fmt.Errorf("types not matched")
}

//NDay is for
func NDay(params ...interface{}) interface{} {
	if len(params) != 2 {
		return fmt.Errorf("params not matched")
	}
	p := params[0]
	//p1 := params[1]
	length := 0.0
	ok := false
	if length, ok = params[1].(float64); ok == false {
		return fmt.Errorf("params not matched")
	}
	switch p.(type) {
	case pie.Float64s:
		p1pie := p.(pie.Float64s)
		ret := pie.Float64s{}
		n := int(length)
		for i := n - 1; i > 0; i-- {
			ret = append(ret, 0.0)
		}
		for i := p1pie.Len() - n; i >= 0; i-- {
			iPos := i
			c := 0
			startPos := iPos + n // n = 1
			if p1pie[iPos] > 0 {
				c = 1
			}
			if p1pie[iPos] < 0 {
				c = -1
			}
			for prevIdx := i + 1; prevIdx < startPos; prevIdx++ {
				if c == 1 && p1pie[prevIdx] < 0 {
					c = 0
				} else if c == -1 && p1pie[prevIdx] > 0 {
					c = 0
				} else {

				}
			}
			ret = pappend(ret, float64(c))
		}
		return ret

	}

	return fmt.Errorf("types not matched")
}

//Barslast is for
func Barslast(params ...interface{}) interface{} {
	if len(params) != 1 {
		return fmt.Errorf("params not matched")
	}
	p := params[0]
	switch p.(type) {
	case pie.Float64s:
		p1pie := p.(pie.Float64s)
		ret := pie.Float64s{}
		for i := p1pie.Len() - 1; i >= 0; i-- {
			c := 0.0
			for sidx := i; sidx < p1pie.Len(); sidx++ {
				if p1pie[sidx] > 0.0 {
					break
				}
				c++
			}
			ret = pappend(ret, c)
		}
		return ret
	}

	return fmt.Errorf("types not matched")

}

//Barssince is for
func Barssince(params ...interface{}) interface{} {
	if len(params) != 1 {
		return fmt.Errorf("params not matched")
	}
	p := params[0]
	switch p.(type) {
	case pie.Float64s:
		p1pie := p.(pie.Float64s)
		ret := pie.Float64s{}
		for i := p1pie.Len() - 1; i >= 0; i-- {
			c := 0.0
			fc := 0.0
			for sidx := i; sidx >= 0; sidx-- {
				if p1pie[sidx] > 0.0 {
					fc = c
				}
				c++
			}
			ret = pappend(ret, fc)
		}
		return ret
	}

	return fmt.Errorf("types not matched")
}

//Barscount is for
func Barscount(params ...interface{}) interface{} {
	if len(params) != 1 {
		return fmt.Errorf("params not matched")
	}
	p := params[0]
	switch p.(type) {
	case pie.Float64s:
		p1pie := p.(pie.Float64s)
		ret := pie.Float64s{}
		for i := p1pie.Len() - 1; i >= 0; i-- {
			ret = append(ret, float64(i))
		}
		return ret
	}

	return fmt.Errorf("types not matched")
}

// CurrentDayBarsCount is for
func CurrentDayBarsCount(params ...interface{}) interface{} {
	if len(params) != 1 {
		return fmt.Errorf("params not matched")
	}
	p := params[0]
	switch p.(type) {
	case []time.Time:
		p1pie := p.([]time.Time)
		ret := pie.Float64s{}
		for i := 1; i < len(p1pie); i++ {
			year, month, day := p1pie[i].Date()
			c := 0.0
			for sidx := i; sidx >= 0; sidx-- {
				pyear, pmonth, pday := p1pie[sidx].Date()

				if year == pyear && month == pmonth && day == pday {
					c++
				} else {
					break
				}
			}
			ret = append(ret, float64(c))
		}
		ret = append(ret, float64(0))
		return ret
	}

	return fmt.Errorf("types not matched")
}

// Year is for
func Year(params ...interface{}) interface{} {
	if len(params) != 1 {
		return fmt.Errorf("params not matched")
	}
	p := params[0]
	switch p.(type) {
	case []time.Time:
		p1pie := p.([]time.Time)
		ret := pie.Float64s{}
		for i := 0; i < len(p1pie); i++ {
			year, _, _ := p1pie[i].Date()
			ret = append(ret, float64(year))
		}
		return ret
	}

	return fmt.Errorf("types not matched")
}

// Month is for
func Month(params ...interface{}) interface{} {
	if len(params) != 1 {
		return fmt.Errorf("params not matched")
	}
	p := params[0]
	switch p.(type) {
	case []time.Time:
		p1pie := p.([]time.Time)
		ret := pie.Float64s{}
		for i := 0; i < len(p1pie); i++ {
			_, month, _ := p1pie[i].Date()
			ret = append(ret, float64(month))
		}
		return ret
	}

	return fmt.Errorf("types not matched")
}

// Day is for
func Day(params ...interface{}) interface{} {
	if len(params) != 1 {
		return fmt.Errorf("params not matched")
	}
	p := params[0]
	switch p.(type) {
	case []time.Time:
		p1pie := p.([]time.Time)
		ret := pie.Float64s{}
		for i := 0; i < len(p1pie); i++ {
			_, _, day := p1pie[i].Date()
			ret = append(ret, float64(day))
		}
		return ret
	}

	return fmt.Errorf("types not matched")
}

//IsLastBar is for
func IsLastBar(params ...interface{}) interface{} {
	if len(params) != 1 {
		return fmt.Errorf("params not matched")
	}
	p := params[0]
	switch p.(type) {
	case pie.Float64s:
		p1pie := p.(pie.Float64s)
		ret := pie.Float64s{}
		for i := p1pie.Len() - 2; i >= 0; i-- {
			ret = append(ret, 0)
		}
		ret = pappend(ret, 1)
		return ret
	}

	return fmt.Errorf("types not matched")
}

// Count is for
func Count(params ...interface{}) interface{} {
	if len(params) != 2 {
		return fmt.Errorf("params not matched")
	}
	p := params[0]
	p1 := params[1]
	switch p.(type) {
	case pie.Float64s:
		switch p1.(type) {
		case float64:
			ret := pie.Float64s{}
			for r := range floatRollingIterator(p.(pie.Float64s)).Iterator(int(p1.(float64))) {
				if r != nil {
					forCount := pie.Float64s(r).Filter(func(v float64) bool {
						return v > 0.0
					})
					ret = append([]float64{float64(forCount.Len())}, ret...)
				} else {
					ret = append(ret, 0.0)
				}
			}
			return ret
		}
	}

	return fmt.Errorf("types not matched")
}

// If is for
func If(params ...interface{}) interface{} {
	if len(params) != 3 {
		return fmt.Errorf("params not matched")
	}
	p := params[0]
	p1 := params[1]
	p2 := params[2]
	var p1pie pie.Float64s

	var ok bool = false
	if p1pie, ok = p.(pie.Float64s); ok == false {
		return fmt.Errorf("types not matched")
	}

	ret := pie.Float64s{}
	for i := 0; i < p1pie.Len(); i++ {
		c := 0.0
		if p1pie[i] > 0.0 {
			switch p1.(type) {
			case float64:
				c = p1.(float64)
			case pie.Float64s:
				c = p1.(pie.Float64s)[i]
			}
		} else {
			switch p2.(type) {
			case float64:
				c = p2.(float64)
			case pie.Float64s:
				c = p2.(pie.Float64s)[i]
			}
		}
		ret = append(ret, c)
	}
	return ret

}

// Ref is for
func Ref(params ...interface{}) interface{} {
	if len(params) != 2 {
		return fmt.Errorf("params not matched")
	}
	p := params[0]
	p1 := params[1]
	//fmt.Println("Ref", p1, p)
	switch p.(type) {
	case pie.Float64s:
		switch p1.(type) {
		case pie.Float64s:
			ret := pie.Float64s{}
			refed := p.(pie.Float64s)
			param := p1.(pie.Float64s)
			for i := 0; i < len(refed); i++ {
				offset := int(param[i])
				// For time-descending data (newest first), REF(C,1) should reference next index (previous time)
				targetIndex := i + offset
				if targetIndex >= len(refed) {
					// When out of bounds, use current value instead of 0
					ret = append(ret, refed[i])
				} else {
					ret = append(ret, refed[targetIndex])
				}
			}
			return ret

		case float64:
			ret := pie.Float64s{}
			param := int(p1.(float64))
			refed := p.(pie.Float64s)

			// For time-descending data (newest first)
			// REF(C,1) means reference previous time period, which is next index in array
			for i := 0; i < len(refed); i++ {
				targetIndex := i + param
				if targetIndex >= len(refed) {
					// When out of bounds, use current value instead of 0
					ret = append(ret, refed[i])
				} else {
					ret = append(ret, refed[targetIndex])
				}
			}
			return ret
		}
	}

	return fmt.Errorf("types not matched")
}

// Llv is for
func Llv(params ...interface{}) interface{} {
	if len(params) != 2 {
		return fmt.Errorf("params not matched")
	}
	p := params[0]
	p1 := params[1]
	//fmt.Println("Llv", p1, p)
	switch p.(type) {
	case pie.Float64s:
		switch p1.(type) {
		case float64:
			p1i := int(p1.(float64))
			ret := pie.Float64s{}
			p1pie := p.(pie.Float64s)
			length := p1pie.Len()
			for i := length - 1; i >= 0; i-- {
				if length-i <= p1i {
					minVal := p1pie[i:].Min()
					ret = pappend(ret, minVal)
				} else {
					minVal := p1pie[i : i+p1i].Min()
					ret = pappend(ret, minVal)
				}
			}
			return ret
		}
	}

	return fmt.Errorf("types not matched")
}

// Hhv is for
func Hhv(params ...interface{}) interface{} {
	if len(params) != 2 {
		return fmt.Errorf("params not matched")
	}
	p := params[0]
	p1 := params[1]
	//fmt.Println("Hhv", p1, p)
	switch p.(type) {
	case pie.Float64s:
		switch p1.(type) {
		case float64:
			p1i := int(p1.(float64))
			ret := pie.Float64s{}
			p1pie := p.(pie.Float64s)
			length := p1pie.Len()
			for i := length - 1; i >= 0; i-- {
				if length-i <= p1i {
					minVal := p1pie[i:].Max()
					ret = pappend(ret, minVal)
				} else {
					minVal := p1pie[i : i+p1i].Max()
					ret = pappend(ret, minVal)
				}
			}
			return ret
		}
	}

	return fmt.Errorf("types not matched")
}

// Not is for
func Not(params ...interface{}) interface{} {
	if len(params) != 1 {
		return fmt.Errorf("params not matched")
	}
	p := params[0]
	switch p.(type) {
	case float64:
		v := int64(p.(float64))
		if v == 0 {
			return float64(1)
		}
		return float64(0)
	case pie.Float64s:
		v := p.(pie.Float64s)
		return v.Map(func(vdo float64) float64 {
			if vdo == 0 {
				return float64(1)
			}
			return float64(0)
		})
	}
	return fmt.Errorf("type not matched")
}

// Minus is for
func Minus(params ...interface{}) interface{} {
	if len(params) == 1 { // -1
		p := params[0]
		switch p.(type) {
		case float64:
			v := p.(float64)
			return -v
		case pie.Float64s:
			v := p.(pie.Float64s)
			return v.Map(func(vdo float64) float64 {
				return -vdo
			})
		}
		return fmt.Errorf("type not matched")

	}
	if len(params) != 2 {
		return fmt.Errorf("params not matched")
	}

	p := params[0]
	p1 := params[1]
	//ln("Minus", p1, p)
	switch p.(type) {
	case float64:
		v := p.(float64)
		switch p1.(type) {
		case float64:
			return v - p1.(float64)
		case pie.Float64s:
			return p1.(pie.Float64s).Map(func(vdo float64) float64 {
				return v - vdo
			})
		}
	case pie.Float64s:
		v := p.(pie.Float64s)
		switch p1.(type) {
		case float64:
			return v.Map(func(vdo float64) float64 {
				return p1.(float64) - vdo
			})
		case pie.Float64s:
			idx := 0
			return v.Map(func(vdo float64) float64 {
				calced := vdo - p1.(pie.Float64s)[idx]
				idx++
				return calced
			})
		}
	}
	return fmt.Errorf("type not matched")
}

// Multiply is for
func Multiply(params ...interface{}) interface{} {
	if len(params) != 2 {
		return fmt.Errorf("params not matched")
	}
	p := params[0]
	p1 := params[1]
	switch p.(type) {
	case float64:
		v := p.(float64)
		switch p1.(type) {
		case float64:
			return v * p1.(float64)
		case pie.Float64s:
			return p1.(pie.Float64s).Map(func(vdo float64) float64 {
				return v * vdo
			})
		}
	case pie.Float64s:
		v := p.(pie.Float64s)
		switch p1.(type) {
		case float64:
			return v.Map(func(vdo float64) float64 {
				return p1.(float64) * vdo
			})
		case pie.Float64s:
			idx := 0
			return v.Map(func(vdo float64) float64 {
				calced := p1.(pie.Float64s)[idx] * vdo
				idx++
				return calced
			})
		}
	}
	return fmt.Errorf("type not matched")
}

// Divide is for
func Divide(params ...interface{}) interface{} {
	if len(params) != 2 {
		return fmt.Errorf("params not matched")
	}
	p := params[0]
	p1 := params[1]
	switch p.(type) {
	case float64:
		v := p.(float64)
		switch p1.(type) {
		case float64:
			return v / p1.(float64)
		case pie.Float64s:
			return p1.(pie.Float64s).Map(func(vdo float64) float64 {
				return v / vdo
			})
		}
	case pie.Float64s:
		v := p.(pie.Float64s)
		switch p1.(type) {
		case float64:
			return v.Map(func(vdo float64) float64 {
				return vdo / p1.(float64)
			})
		case pie.Float64s:
			idx := 0
			return v.Map(func(vdo float64) float64 {
				calced := vdo / p1.(pie.Float64s)[idx]
				idx++
				return calced
			})
		}
	}
	return fmt.Errorf("type not matched")
}

// Add is for
func Add(params ...interface{}) interface{} {
	if len(params) != 2 {
		return fmt.Errorf("params not matched")
	}
	p := params[0]
	p1 := params[1]
	//fmt.Println("Add", p1, p)
	switch p.(type) {
	case float64:
		v := p.(float64)
		switch p1.(type) {
		case float64:
			return v + p1.(float64)
		case pie.Float64s:
			return p1.(pie.Float64s).Map(func(vdo float64) float64 {
				return v + vdo
			})
		}
	case pie.Float64s:
		v := p.(pie.Float64s)
		switch p1.(type) {
		case float64:
			return v.Map(func(vdo float64) float64 {
				return vdo + p1.(float64)
			})
		case pie.Float64s:
			idx := 0
			return v.Map(func(vdo float64) float64 {
				calced := vdo + p1.(pie.Float64s)[idx]
				idx++
				return calced
			})
		}
	}
	return fmt.Errorf("type not matched")
}

// Eq is for
func Eq(params ...interface{}) interface{} {
	if len(params) != 2 {
		return fmt.Errorf("params not matched")
	}
	p := params[0]
	p1 := params[1]
	switch p.(type) {
	case float64:
		v := p.(float64)
		switch p1.(type) {
		case float64:
			return almostEqual(v, p1.(float64))
		case pie.Float64s:
			return p1.(pie.Float64s).Map(func(vdo float64) float64 {
				if almostEqual(v, p1.(float64)) == true {
					return 1.0
				} else {
					return 0.0
				}
			})
		}
	case pie.Float64s:
		v := p.(pie.Float64s)
		switch p1.(type) {
		case float64:
			return v.Map(func(vdo float64) float64 {

				if almostEqual(vdo, p1.(float64)) == true {
					return 1.0
				} else {
					return 0.0
				}
			})
		case pie.Float64s:
			idx := 0
			return v.Map(func(vdo float64) float64 {
				calced := almostEqual(vdo, p.(pie.Float64s)[idx])
				idx++
				if calced == true {
					return 1.0
				} else {
					return 0.0
				}
			})
		}
	}
	return fmt.Errorf("type not matched")
}

// Neq is for
func Neq(params ...interface{}) interface{} {
	if len(params) != 2 {
		return fmt.Errorf("params not matched")
	}
	p := params[0]
	p1 := params[1]
	switch p.(type) {
	case float64:
		v := p.(float64)
		switch p1.(type) {
		case float64:
			return !almostEqual(v, p1.(float64))
		case pie.Float64s:

			ret := p1.(pie.Float64s).Map(func(vdo float64) float64 {
				if !almostEqual(v, p1.(float64)) == true {
					return 1.0
				} else {
					return 0.0
				}
			})
			log.Printf("funcs: Neq ret=%v", ret)
			return ret
		}
	case pie.Float64s:
		v := p.(pie.Float64s)
		switch p1.(type) {
		case float64:
			return v.Map(func(vdo float64) float64 {
				if !almostEqual(vdo, p1.(float64)) == true {
					return 1.0
				} else {
					return 0.0
				}
			})
		case pie.Float64s:
			idx := 0
			return v.Map(func(vdo float64) float64 {
				calced := !almostEqual(vdo, p.(pie.Float64s)[idx])
				idx++
				if calced == true {
					return 1.0
				} else {
					return 0.0
				}
			})
		}
	}
	return fmt.Errorf("type not matched")
}

// Gt is for
func Gt(params ...interface{}) interface{} {
	if len(params) != 2 {
		return fmt.Errorf("params not matched")
	}
	p := params[0]
	p1 := params[1]
	switch p.(type) {
	case float64:
		v := p.(float64)
		switch p1.(type) {
		case float64:
			return v > p1.(float64)
		case pie.Float64s:
			return p1.(pie.Float64s).Map(func(vdo float64) float64 {
				if v > p1.(float64) {
					return 1.0
				} else {
					return 0.0
				}
			})
		}
	case pie.Float64s:
		v := p.(pie.Float64s)
		switch p1.(type) {
		case float64:
			return v.Map(func(vdo float64) float64 {
				if vdo > p1.(float64) {
					return 1.0
				} else {
					return 0.0
				}
			})

		case pie.Float64s:
			idx := 0

			ret := v.Map(func(vdo float64) float64 {
				calced := vdo > p1.(pie.Float64s)[idx]
				idx++
				if calced == true {
					return 1.0
				} else {
					return 0.0
				}
			})

			return ret
		}
	}
	return fmt.Errorf("type not matched")
}

// Gte is for
func Gte(params ...interface{}) interface{} {
	if len(params) != 2 {
		return fmt.Errorf("params not matched")
	}
	p := params[0]
	p1 := params[1]
	switch p.(type) {
	case float64:
		v := p.(float64)
		switch p1.(type) {
		case float64:
			return v >= p1.(float64)
		case pie.Float64s:
			return p1.(pie.Float64s).Map(func(vdo float64) float64 {
				if v >= p1.(float64) {
					return 1.0
				} else {
					return 0.0
				}
			})
		}
	case pie.Float64s:
		v := p.(pie.Float64s)
		switch p1.(type) {
		case float64:
			return v.Map(func(vdo float64) float64 {

				if vdo >= p1.(float64) {
					return 1.0
				} else {
					return 0.0
				}
			})
		case pie.Float64s:
			idx := 0
			return v.Map(func(vdo float64) float64 {
				calced := vdo >= p.(pie.Float64s)[idx]
				idx++
				if calced == true {
					return 1.0
				} else {
					return 0.0
				}
			})
		}
	}
	return fmt.Errorf("type not matched")
}

// Lte is for
func Lte(params ...interface{}) interface{} {
	if len(params) != 2 {
		return fmt.Errorf("params not matched")
	}
	p := params[0]
	p1 := params[1]
	switch p.(type) {
	case float64:
		v := p.(float64)
		switch p1.(type) {
		case float64:
			return v <= p1.(float64)
		case pie.Float64s:
			return p1.(pie.Float64s).Map(func(vdo float64) float64 {
				if v <= p1.(float64) {
					return 1.0
				} else {
					return 0.0
				}
			})
		}
	case pie.Float64s:
		v := p.(pie.Float64s)
		switch p1.(type) {
		case float64:
			return v.Map(func(vdo float64) float64 {

				if vdo <= p1.(float64) {
					return 1.0
				} else {
					return 0.0
				}
			})
		case pie.Float64s:
			idx := 0
			return v.Map(func(vdo float64) float64 {
				calced := vdo <= p.(pie.Float64s)[idx]
				idx++
				if calced == true {
					return 1.0
				} else {
					return 0.0
				}
			})
		}
	}
	return fmt.Errorf("type not matched")
}

// Lt is for
func Lt(params ...interface{}) interface{} {
	if len(params) != 2 {
		return fmt.Errorf("params not matched")
	}
	p := params[0]
	p1 := params[1]
	switch p.(type) {
	case float64:
		v := p.(float64)
		switch p1.(type) {
		case float64:
			return v < p1.(float64)
		case pie.Float64s:
			return p1.(pie.Float64s).Map(func(vdo float64) float64 {
				if v < p1.(float64) {
					return 1.0
				} else {
					return 0.0
				}
			})
		}
	case pie.Float64s:
		v := p.(pie.Float64s)
		switch p1.(type) {
		case float64:
			return v.Map(func(vdo float64) float64 {

				if vdo < p1.(float64) {
					return 1.0
				} else {
					return 0.0
				}
			})
		case pie.Float64s:
			idx := 0
			return v.Map(func(vdo float64) float64 {
				calced := vdo < p.(pie.Float64s)[idx]
				idx++
				if calced == true {
					return 1.0
				} else {
					return 0.0
				}
			})
		}
	}
	return fmt.Errorf("type not matched")
}

// And is for
func And(params ...interface{}) interface{} {
	if len(params) != 2 {
		return fmt.Errorf("params not matched")
	}
	p := params[0]
	p1 := params[1]
	switch p.(type) {
	case float64:
		v := p.(float64)
		switch p1.(type) {
		case float64:
			if (v > 0.0) && (p1.(float64) > 0.0) {
				return 1.0
			}
			return 0.0
		case pie.Float64s:
			return p1.(pie.Float64s).Map(func(vdo float64) float64 {
				if vdo > 0.0 && v > 0.0 {
					return 1.0
				}
				return 0.0
			})
		}
	case pie.Float64s:
		v := p.(pie.Float64s)
		switch p1.(type) {
		case float64:
			v1 := p1.(float64)
			return v.Map(func(vdo float64) float64 {
				if vdo > 0.0 && v1 > 0.0 {
					return 1.0
				}
				return 0.0
			})
		case pie.Float64s:
			idx := 0
			return v.Map(func(vdo float64) float64 {
				calced := vdo > 0.0 && p.(pie.Float64s)[idx] > 0.0
				idx++
				if calced == true {
					return 1.0
				} else {
					return 0.0
				}
			})
		}
	}
	return fmt.Errorf("type not matched")
}

// Or is for
func Or(params ...interface{}) interface{} {
	if len(params) != 2 {
		return fmt.Errorf("params not matched")
	}
	p := params[0]
	p1 := params[1]
	switch p.(type) {
	case float64:
		v := p.(float64)
		switch p1.(type) {
		case float64:
			if (v > 0.0) || (p1.(float64) > 0.0) {
				return 1.0
			}
			return 0.0
		case pie.Float64s:
			return p1.(pie.Float64s).Map(func(vdo float64) float64 {
				if vdo > 0.0 || v > 0.0 {
					return 1.0
				}
				return 0.0
			})
		}
	case pie.Float64s:
		v := p.(pie.Float64s)
		switch p1.(type) {
		case float64:
			v1 := p1.(float64)
			return v.Map(func(vdo float64) float64 {
				if vdo > 0.0 || v1 > 0.0 {
					return 1.0
				}
				return 0.0
			})
		case pie.Float64s:
			idx := 0
			return v.Map(func(vdo float64) float64 {
				calced := vdo > 0.0 || p.(pie.Float64s)[idx] > 0.0
				idx++
				if calced == true {
					return 1.0
				} else {
					return 0.0
				}
			})
		}
	}
	return fmt.Errorf("type not matched")
}
