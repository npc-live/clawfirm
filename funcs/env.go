package funcs

import (
	"log"

	"github.com/alecthomas/participle/v2"
	"github.com/elliotchance/pie/pie"
)

// EvalContext is for
type EvalContext struct {
	Values map[string]interface{}
	Funcs  map[string]HandlerFunc
	Parser *participle.Parser
}

func importLibrary(ctx *EvalContext) {
	ctx.Funcs["abs"] = Abs
	ctx.Funcs["sum"] = Sum
	ctx.Funcs["stddev"] = Stddev
	ctx.Funcs["sma"] = Sma
	ctx.Funcs["ema"] = Ema
	ctx.Funcs["ma"] = Ma
	ctx.Funcs["max"] = Max
	ctx.Funcs["min"] = Min

	ctx.Funcs["slope"] = Slope

	ctx.Funcs["cross"] = Cross
	ctx.Funcs["barssince"] = Barssince
	ctx.Funcs["barslast"] = Barslast
	ctx.Funcs["barscount"] = Barscount
	ctx.Funcs["count"] = Count
	ctx.Funcs["if"] = If
	ctx.Funcs["ref"] = Ref
	ctx.Funcs["llv"] = Llv
	ctx.Funcs["hhv"] = Hhv
	ctx.Funcs["not"] = Not
	ctx.Funcs["corr"] = Corr
	ctx.Funcs["ABS"] = Abs
	ctx.Funcs["SUM"] = Sum
	ctx.Funcs["STDDEV"] = Stddev
	ctx.Funcs["SMA"] = Sma
	ctx.Funcs["EMA"] = Ema
	ctx.Funcs["MA"] = Ma
	ctx.Funcs["MAX"] = Max
	ctx.Funcs["MIN"] = Min
	ctx.Funcs["SLOPE"] = Slope
	ctx.Funcs["CROSS"] = Cross
	ctx.Funcs["BARSSINCE"] = Barssince
	ctx.Funcs["BARSLAST"] = Barslast
	ctx.Funcs["BARSCOUNT"] = Barscount
	ctx.Funcs["COUNT"] = Count
	ctx.Funcs["IF"] = If
	ctx.Funcs["REF"] = Ref
	ctx.Funcs["LLV"] = Llv
	ctx.Funcs["HHV"] = Hhv
	ctx.Funcs["NOT"] = Not
	ctx.Funcs["CORR"] = Corr

	ctx.Funcs["minus"] = Minus
	ctx.Funcs["multiply"] = Multiply
	ctx.Funcs["divide"] = Divide
	ctx.Funcs["add"] = Add
	ctx.Funcs[">="] = Gte
	ctx.Funcs["=="] = Eq
	ctx.Funcs["!="] = Neq
	ctx.Funcs[">"] = Gt
	ctx.Funcs["<"] = Lt
	ctx.Funcs["<="] = Lte
	ctx.Funcs["&&"] = And
	ctx.Funcs["||"] = Or
	ctx.Funcs["*"] = Multiply
	ctx.Funcs["/"] = Divide
	ctx.Funcs["+"] = Add
	ctx.Funcs["-"] = Minus
}

//New is for
func New(vals map[string]interface{}) *EvalContext {
	//defer profiling.TimeTrack(time.Now(), "Indicator New")
	ctx := EvalContext{
		Funcs:  make(map[string]HandlerFunc, 100),
		Values: make(map[string]interface{}, 100), // 增加容量到100
		Parser: Compile(),
	}
	importLibrary(&ctx)

	// 处理传入的数据
	for k, v := range vals {
		if k == "list" {
			// 如果是list数据，需要转换为OHLC数组
			if listData, ok := v.([]interface{}); ok {
				o := pie.Float64s{}
				c := pie.Float64s{}
				h := pie.Float64s{}
				l := pie.Float64s{}

				for _, item := range listData {
					if kline, ok := item.(map[string]interface{}); ok {
						if open, ok := kline["open"].(float64); ok {
							o = append(o, open)
						}
						if close, ok := kline["close"].(float64); ok {
							c = append(c, close)
						}
						if high, ok := kline["high"].(float64); ok {
							h = append(h, high)
						}
						if low, ok := kline["low"].(float64); ok {
							l = append(l, low)
						}
					}
				}

				// 设置OHLC数组（支持大小写）
				ctx.Values["o"] = o
				ctx.Values["c"] = c
				ctx.Values["h"] = h
				ctx.Values["l"] = l
				ctx.Values["O"] = o
				ctx.Values["C"] = c
				ctx.Values["H"] = h
				ctx.Values["L"] = l
			}
		} else {
			// 其他数据直接复制
			ctx.Values[k] = v
		}
	}
	return &ctx
}

func (ctx *EvalContext) Eval(indicator string) {
	//defer profiling.TimeTrack(time.Now(), "Indicator Eval")
	ast := &PStmts{}
	err := ctx.Parser.ParseString("indicator", indicator, ast)
	if err != nil {
		log.Printf("funcs: Eval error: %v", err)
		//fmt.Println(err)
		return
	}
	ast.Eval(ctx)
}

//
