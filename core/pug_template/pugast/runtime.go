package pugast

import (
	"encoding/json"
	"fmt"
	"html/template"
	"reflect"
	"strconv"
	"strings"
)

// FuncMap is the default runtime funcmap for pugast templates
var funcmap = FuncMap{
	"__": func(s ...string) string { return strings.Join(s, "::") },

	"__op__add":   runtimeAdd,
	"__op__inc":   runtimeInc,
	"__op__sub":   runtimeSub,
	"__op__mul":   runtimeMul,
	"__op__quo":   runtimeQuo,
	"__op__slash": runtimeQuo,
	"__op__rem":   runtimeRem,
	"__op__mod":   runtimeRem,
	"__op__minus": runtimeMinus,
	"__op__plus":  runtimePlus,
	"__op__eql":   runtimeEql,
	"__op__gtr":   runtimeGtr,
	"__op__lss":   runtimeLss,
	"neq":         func(x, y interface{}) bool { return !runtimeEql(x, y) },

	"tryindex": func(obj, key interface{}) interface{} {
		//log.Println(obj, key)
		arr, ok := obj.(*Array)
		idx, ok2 := key.(int)
		if ok && ok2 {
			return arr.items[idx]
		}

		if obj, ok := obj.(Object); ok {
			return obj.Field(convert(key).String())
		}

		vo, _ := indirect(reflect.ValueOf(obj))
		k := int(reflect.ValueOf(key).Int())
		if !vo.IsValid() {
			return nil
		}
		if vo.Len() > k {
			return vo.Index(k).Interface()
		}
		return nil
	},

	"json":      runtimeJSON,
	"unescaped": runtimeUnescaped,

	"null": func() interface{} { return nil },

	"_Range": func(args ...int) Object {
		var res []int
		var m, o int
		if len(args) == 1 {
			m = args[0]
			o = 0
		} else {
			m = args[1]
			o = args[0]
		}

		for i := o; i < m; i++ {
			res = append(res, i)
		}
		return convert(res)
	},

	"raw":     func(s ...interface{}) template.HTML { return template.HTML(fmt.Sprint(s...)) },
	"tagopen": func(t, p string) template.HTML { return template.HTML(`<` + p + t) },
	"s": func(l ...interface{}) string {
		var res string
		for _, s := range l {
			if s != nil {
				res += convert(s).String()
			}
			//vs := reflect.ValueOf(s)
			//switch vs.Kind() {
			//case reflect.Int, reflect.Int32, reflect.Int64, reflect.Int16, reflect.Int8:
			//	{
			//		res += fmt.Sprintf("%d", vs.Int())
			//	}
			//case reflect.Float32, reflect.Float64:
			//	{
			//		res += fmt.Sprintf("%f", vs.Float())
			//	}
			//case reflect.String:
			//	{
			//		res += vs.String()
			//	}
			//}
		}
		if len(res) > 1 {
			return " " + strings.TrimSpace(res)
		}
		return ""
	},
	"sc": func(l ...interface{}) (res template.CSS) {
		for _, s := range l {
			vs := reflect.ValueOf(s)
			switch vs.Kind() {
			case reflect.Int, reflect.Int32, reflect.Int64, reflect.Int16, reflect.Int8:
				{
					res += template.CSS(fmt.Sprintf("%d", vs.Int()))
				}
			case reflect.Float32, reflect.Float64:
				{
					res += template.CSS(fmt.Sprintf("%f", vs.Float()))
				}
			case reflect.String:
				{
					res += template.CSS(vs.String())
				}
			}
		}
		return
	},

	"__op__array": func(a ...interface{}) Object { return convert(a) },
	"__op__map": func(a ...interface{}) Object {
		m := make(map[interface{}]interface{}, len(a)/2)
		for i := 0; i < len(a); i += 2 {
			m[a[i]] = a[i+1]
		}
		return convert(m)
	},
	"attr": func(attr, prefix interface{}) string {
		if attr == nil {
			return ""
		}
		t := strings.Split(attr.(string), " ")
		for k, v := range t {
			t[k] = prefix.(string) + "-" + v
		}
		return strings.Join(t, " ")
	},
	"extend": func(deep bool, m map[interface{}]interface{}, on ...map[interface{}]interface{}) map[interface{}]interface{} {
		for _, o := range on {
			for k, v := range o {
				m[k] = v
			}
		}
		return m
	},

	"__add_andattributes": func(attrs Object, k ...string) template.HTMLAttr {
		known := make(map[string]bool)
		for _, k := range k {
			known[k] = true
		}
		res := ""
		if attrs, ok := attrs.(*Map); ok && attrs.Items != nil {
			for k, v := range attrs.Items {
				if !known[k.String()] {
					res += ` ` + k.String() + `="` + strings.TrimSpace(v.String()) + `"`
				}
			}
		}
		return template.HTMLAttr(strings.TrimSpace(res))
	},
}

func runtimeAdd(l, r interface{}) Object {
	x := convert(l)
	y := convert(r)

	switch x := x.(type) {
	case String:
		return String(x.String() + y.String())

	case Number:
		switch y := y.(type) {
		case Number:
			return Number(x + y)

		case String:
			f, _ := strconv.ParseFloat(y.String(), 64)
			return Number(float64(x) + f)
		}
	}
	return Nil{}
}

func runtimeInc(x interface{}) int {
	vx := reflect.ValueOf(x)
	switch vx.Kind() {
	case reflect.Int, reflect.Int32, reflect.Int64, reflect.Int16, reflect.Int8:
		{
			return int(vx.Int() + 1)
		}
	case reflect.Float32, reflect.Float64:
		{
			return int(vx.Float() + 1)
		}
	}

	return 0
}

func runtimeSub(i ...interface{}) interface{} {
	y := i[0]
	var x interface{}
	if len(i) > 1 {
		x = i[1]
	} else {
		x = 0
	}
	vx, vy := reflect.ValueOf(x), reflect.ValueOf(y)
	switch vx.Kind() {
	case reflect.Int, reflect.Int32, reflect.Int64, reflect.Int16, reflect.Int8:
		{
			switch vy.Kind() {
			case reflect.Int, reflect.Int32, reflect.Int64, reflect.Int16, reflect.Int8:
				return int(vx.Int() - vy.Int())
			case reflect.Float32, reflect.Float64:
				return float64(vx.Int()) - vy.Float()
			}
		}
	case reflect.Float32, reflect.Float64:
		{
			switch vy.Kind() {
			case reflect.Int, reflect.Int32, reflect.Int64, reflect.Int16, reflect.Int8:
				return vx.Float() - float64(vy.Int())
			case reflect.Float32, reflect.Float64:
				return vx.Float() - vy.Float()
			}
		}
	}

	return "<nil>"
}

func runtimeMul(x, y interface{}) interface{} {
	vx, vy := reflect.ValueOf(x), reflect.ValueOf(y)
	switch vx.Kind() {
	case reflect.Int, reflect.Int32, reflect.Int64, reflect.Int16, reflect.Int8:
		{
			switch vy.Kind() {
			case reflect.Int, reflect.Int32, reflect.Int64, reflect.Int16, reflect.Int8:
				return int(vx.Int() * vy.Int())
			case reflect.Float32, reflect.Float64:
				return float64(vx.Int()) * vy.Float()
			}
		}
	case reflect.Float32, reflect.Float64:
		{
			switch vy.Kind() {
			case reflect.Int, reflect.Int32, reflect.Int64, reflect.Int16, reflect.Int8:
				return vx.Float() * float64(vy.Int())
			case reflect.Float32, reflect.Float64:
				return vx.Float() * vy.Float()
			}
		}
	}

	return "<nil>"
}

func runtimeQuo(x, y interface{}) interface{} {
	vx, vy := reflect.ValueOf(x), reflect.ValueOf(y)
	switch vx.Kind() {
	case reflect.Int, reflect.Int32, reflect.Int64, reflect.Int16, reflect.Int8:
		{
			switch vy.Kind() {
			case reflect.Int, reflect.Int32, reflect.Int64, reflect.Int16, reflect.Int8:
				return float64(vx.Int()) / float64(vy.Int())
			case reflect.Float32, reflect.Float64:
				return float64(vx.Int()) / vy.Float()
			}
		}
	case reflect.Float32, reflect.Float64:
		{
			switch vy.Kind() {
			case reflect.Int, reflect.Int32, reflect.Int64, reflect.Int16, reflect.Int8:
				return vx.Float() / float64(vy.Int())
			case reflect.Float32, reflect.Float64:
				return vx.Float() / vy.Float()
			}
		}
	}

	return "<nil>"
}

func runtimeRem(x, y interface{}) interface{} {
	vx, vy := reflect.ValueOf(x), reflect.ValueOf(y)
	switch vx.Kind() {
	case reflect.Int, reflect.Int32, reflect.Int64, reflect.Int16, reflect.Int8:
		{
			switch vy.Kind() {
			case reflect.Int, reflect.Int32, reflect.Int64, reflect.Int16, reflect.Int8:
				return int(vx.Int() % vy.Int())
			}
		}
	}

	return "<nil>"
}

func runtimeMinus(x interface{}) interface{} {
	vx := reflect.ValueOf(x)
	switch vx.Kind() {
	case reflect.Int, reflect.Int32, reflect.Int64, reflect.Int16, reflect.Int8:
		return -vx.Int()
	case reflect.Float32, reflect.Float64:
		return -vx.Float()
	}

	return "<nil>"
}

func runtimePlus(x interface{}) interface{} {
	vx := reflect.ValueOf(x)
	switch vx.Kind() {
	case reflect.Int, reflect.Int32, reflect.Int64, reflect.Int16, reflect.Int8:
		return int(+vx.Int())
	case reflect.Float32, reflect.Float64:
		return +vx.Float()
	}

	return "<nil>"
}

func runtimeEql(x, y interface{}) bool {
	vx, vy := reflect.ValueOf(x), reflect.ValueOf(y)
	switch vx.Kind() {
	case reflect.Int, reflect.Int32, reflect.Int64, reflect.Int16, reflect.Int8:
		{
			switch vy.Kind() {
			case reflect.Int, reflect.Int32, reflect.Int64, reflect.Int16, reflect.Int8:
				return vx.Int() == vy.Int()
			case reflect.Float32, reflect.Float64:
				return float64(vx.Int()) == vy.Float()
			case reflect.String:
				return fmt.Sprintf("%d", vx.Int()) == vy.String()
			}
		}
	case reflect.Float32, reflect.Float64:
		{
			switch vy.Kind() {
			case reflect.Int, reflect.Int32, reflect.Int64, reflect.Int16, reflect.Int8:
				return vx.Float() == float64(vy.Int())
			case reflect.Float32, reflect.Float64:
				return vx.Float() == vy.Float()
			case reflect.String:
				return fmt.Sprintf("%f", vx.Float()) == vy.String()
			}
		}
	case reflect.String:
		{
			switch vy.Kind() {
			case reflect.Int, reflect.Int32, reflect.Int64, reflect.Int16, reflect.Int8:
				return vx.String() == fmt.Sprintf("%d", vy.Int())
			case reflect.Float32, reflect.Float64:
				return vx.String() == fmt.Sprintf("%f", vy.Float())
			case reflect.String:
				return vx.String() == fmt.Sprintf("%s", vy.String())
			}
		}
	case reflect.Bool:
		{
			switch vy.Kind() {
			case reflect.Int, reflect.Int32, reflect.Int64, reflect.Int16, reflect.Int8:
				return vx.Bool() && vy.Int() != 0
			case reflect.Bool:
				return vx.Bool() == vy.Bool()
			}
		}
	}

	return false
}

func runtimeLss(x, y interface{}) bool {
	vx, vy := reflect.ValueOf(x), reflect.ValueOf(y)
	switch vx.Kind() {
	case reflect.Int, reflect.Int32, reflect.Int64, reflect.Int16, reflect.Int8:
		{
			switch vy.Kind() {
			case reflect.Int, reflect.Int32, reflect.Int64, reflect.Int16, reflect.Int8:
				return vx.Int() < vy.Int()
			case reflect.Float32, reflect.Float64:
				return float64(vx.Int()) < vy.Float()
			case reflect.String:
				return fmt.Sprintf("%d", vx.Int()) < vy.String()
			}
		}
	case reflect.Float32, reflect.Float64:
		{
			switch vy.Kind() {
			case reflect.Int, reflect.Int32, reflect.Int64, reflect.Int16, reflect.Int8:
				return vx.Float() < float64(vy.Int())
			case reflect.Float32, reflect.Float64:
				return vx.Float() < vy.Float()
			case reflect.String:
				return fmt.Sprintf("%f", vx.Float()) < vy.String()
			}
		}
	case reflect.String:
		{
			switch vy.Kind() {
			case reflect.Int, reflect.Int32, reflect.Int64, reflect.Int16, reflect.Int8:
				return vx.String() < fmt.Sprintf("%d", vy.Int())
			case reflect.Float32, reflect.Float64:
				return vx.String() < fmt.Sprintf("%f", vy.Float())
			case reflect.String:
				return vx.String() < vy.String()
			}
		}
	}

	return false
}

func runtimeGtr(x, y interface{}) bool {
	return !runtimeLss(x, y) && !runtimeEql(x, y)
}

func runtimeJSON(x interface{}) (res template.JS, err error) {
	bres, err := json.Marshal(x)
	res = template.JS(string(bres))
	return
}

func runtimeUnescaped(x string) interface{} {
	return template.HTML(x)
}
