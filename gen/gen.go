package gen

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/printer"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"text/template"
	"unicode"

	"github.com/samber/lo"
)

const (
	target  = "./gen/"
	pkgPath = "go/demo"
)

//go:embed template/*
var tplDir embed.FS

type relation struct {
	Field, Target, First, Second string

	FirstField, SecondField string
	FirstType, SecondType   string

	Slice bool
}

type fieldDefault struct {
	Value   any
	Builtin bool
	Initial string
	Trigger string
}

type field struct {
	Name, Column, Type string

	Default *fieldDefault
}

type modelPK struct {
	Name, Type string
	Int, Auto  bool
}

type model struct {
	Name string
	PK   *modelPK

	Valid     bool
	Imports   []string
	Fields    []field
	Relations map[string]*relation
	Validates []string
}

func (m *model) AddImport(s string) {
	if s == "" || lo.Contains(m.Imports, s) {
		return
	}

	if strings.HasPrefix(s, `"github.com/maxshaw/`) {
		return
	}

	m.Imports = append(m.Imports, s)
}

func isNumPK(typ string) bool {
	switch typ {
	case "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64":
		return true
	}
	return false
}

func newModel(imports []*ast.ImportSpec, name string) *model {
	m := &model{Valid: false, Name: name, Relations: make(map[string]*relation)}
	for _, pkg := range imports {
		m.AddImport(pkg.Path.Value)
	}
	return m
}

func newRelation(field, v string, typ ast.Expr, slice bool) *relation {
	target, ok := typ.(*ast.StarExpr)
	if !ok {
		if slice, ok := typ.(*ast.ArrayType); ok {
			return newRelation(field, v, slice.Elt, true)
		}
		return nil
	}

	rel := relation{
		Field:  field,
		Target: target.X.(*ast.Ident).Name,
		Slice:  slice,
	}

	parts := strings.SplitN(v, ",", 2)
	if len(parts) > 1 {
		rel.First, rel.Second = parts[0], parts[1]
	} else {
		rel.First, rel.Second = parts[0], parts[0]
	}

	return &rel
}

func checkValue(typ string) (string, string) {
	switch typ {
	case "string":
		return ` == ""`, "must be not empty"

	case "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64", "float32", "float64":
		return " == 0", "must be not zero"

	case "time.Time", "*time.Time":
		return ".IsZero()", "not a valid time"

	default:
		if strings.HasPrefix(typ, "*") {
			return " == nil", "must be not null"
		}

		fmt.Printf("[typ] %s\n", typ)
	}

	return "", ""
}

func initialValue(typ, val string) string {
	switch typ {
	case "string":
		return `"` + strings.Trim(val, `"`) + `"`

	case "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64", "float32", "float64":
		return val

	case "time.Time", "*time.Time":
		return val

	default:
		if strings.HasPrefix(typ, "*") {
			return val
		}

		return ""
	}
}

var fset = token.NewFileSet()

func Gen() error {
	t, err := template.New("gen").Funcs(template.FuncMap{
		"lowerField": lowerField,
		"lowerFirst": lowerFirst,
		"join":       strings.Join,
	}).ParseFS(tplDir, "template/*.tmpl")
	if err != nil {
		return err
	}

	files, err := filepath.Glob("./internal/model/*.go")
	if err != nil || len(files) == 0 {
		return errors.New("model folder is empty")
	}

	if stat, err := os.Stat(target); os.IsNotExist(err) || !stat.IsDir() {
		_ = os.Mkdir("./gen", 0777)
	}

	models := make(map[string]*model)
	for _, file := range files {
		src, err := os.ReadFile(file)
		if err != nil {
			return err
		}

		f, err := parser.ParseFile(fset, file, string(src), parser.ParseComments)
		if err != nil {
			return err
		}

		for _, decl := range f.Decls {
			switch d := decl.(type) {
			case *ast.GenDecl:
				for _, spec := range d.Specs {
					if ts, ok := spec.(*ast.TypeSpec); ok {
						if st, ok := ts.Type.(*ast.StructType); ok {
							models[ts.Name.Name] = parse(newModel(f.Imports, ts.Name.Name), st)
						}
					}
				}

			case *ast.FuncDecl:
				if d.Name.Name == "TableName" && firstFieldName(d.Type.Results) == "string" {
					modelName := firstFieldName(d.Recv)
					for mn, m := range models {
						if mn == modelName {
							m.Valid = true
							break
						}
					}
				}
			}
		}
	}

	var names []string
	for mn, model := range models {
		if !model.Valid {
			delete(models, mn)
			continue
		}

		for _, rel := range model.Relations {
			if rel.FirstField == "" || rel.SecondField == "" {
				if exist, ok := lo.Find(model.Fields, func(f field) bool { return f.Column == rel.First }); ok {
					rel.FirstField = exist.Name
					rel.FirstType = exist.Type
				} else {
					return fmt.Errorf("["+mn+"."+rel.Field+"] the first relation column is not exists: %s", rel.First)
				}

				if exist, ok := lo.Find(models[rel.Target].Fields, func(f field) bool { return f.Column == rel.Second }); ok {
					rel.SecondField = exist.Name
					rel.SecondType = exist.Type
				} else {
					return fmt.Errorf("["+mn+"."+rel.Field+"] the second relation column is not exists: %s", rel.Second)
				}

				if rel.FirstType != rel.SecondType {
					return errors.New("[" + mn + "." + rel.Field + "] the first and second field type mismatch: " + rel.FirstField + " -> " + rel.FirstType + ", " + rel.SecondField + " -> " + rel.SecondType)
				}
			}
		}

		names = append(names, mn)
	}

	execTpl(t, "client", "client", map[string]any{
		"PkgPath": pkgPath,
		"Models":  names,
	})

	for _, model := range models {
		cols := lo.Map(model.Fields, func(f field, _ int) string { return f.Column })

		name := strings.ToLower(model.Name)

		execTpl(t, name, "model", map[string]any{
			"PkgPath":   pkgPath,
			"Name":      model.Name,
			"LowerName": lowerFirst(model.Name),
			"Model":     model,
			"Columns":   cols,
			"Select":    strings.Join(cols, `", "`),
		})

		execTpl(t, name+"query", "query", map[string]any{
			"PkgPath":   pkgPath,
			"Name":      model.Name,
			"LowerName": lowerFirst(model.Name),
			"Model":     model,
			"Columns":   cols,
			"Select":    strings.Join(cols, `", "`),
		})

		execTpl(t, name+"update", "update", map[string]any{
			"PkgPath":   pkgPath,
			"Name":      model.Name,
			"LowerName": lowerFirst(model.Name),
			"Model":     model,
			"Columns":   cols,
			"Select":    strings.Join(cols, `", "`),
		})
	}

	return nil
}

func execTpl(t *template.Template, name, tpl string, args any) {
	var out bytes.Buffer
	t.ExecuteTemplate(&out, tpl+".tmpl", args)

	src := out.Bytes()

	b, err := format.Source(src)
	if err != nil {
		fmt.Println(string(src))
		log.Fatal(err)
	}

	if err = os.WriteFile(target+name+".go", b, 0777); err != nil {
		log.Fatal(err)
	}
}

func parse(m *model, st *ast.StructType) *model {
FL:
	for _, sf := range st.Fields.List {
		if len(sf.Names) < 1 {
			continue
		}

		f := field{Name: sf.Names[0].Name, Type: typeName(m, sf.Type, false)}

		if sf.Tag != nil {
			if tag := reflect.StructTag(strings.Trim(sf.Tag.Value, "`")).Get("db"); tag == "-" {
				continue
			} else {
				for _, part := range strings.Split(tag, ";") {
					if !strings.Contains(part, "=") {
						f.Column = part
						continue
					}

					var k, v string
					if kv := strings.SplitN(part, "=", 2); len(kv) == 2 {
						k, v = kv[0], kv[1]
					} else {
						k = kv[0]
					}

					switch k {
					case "pk":
						m.PK = &modelPK{Name: f.Name, Type: f.Type, Auto: v == "auto"}

					case "valid":

						tpl := `
{{define "required"}}
if {{ .Field }}{{ .Required.Val }} {
    return &orm.ValidationError{Field: "{{ .Name }}", Msg: "{{ .Required.Msg }}"}
}
{{end}}

{{define "min"}}
if {{ .Field }} < {{ .Value }} {
    return &orm.ValidationError{Field: "{{ .Name }}", Msg: "must not less than {{ .Value }}"}
}
{{end}}

{{define "max"}}
if {{ .Field }} > {{ .Value }} {
    return &orm.ValidationError{Field: "{{ .Name }}", Msg: "must not greater than {{ .Value }}"}
}
{{end}}

{{define "in"}}
if {{range $i, $v := .Value}}{{if gt $i 0}} && {{end}}{{ $.UnwrappedField }} != {{ $v }}{{end}} {
    return &orm.ValidationError{Field: "{{ .Name }}", Msg: "must be one of {{ .Value1 }}"}
}
{{end}}`

						t := template.Must(template.New("").Parse(tpl))

						type rule struct {
							Name   string
							Value  any
							Value1 any
						}

						var (
							rules    []rule
							required = true
						)

					F:
						for _, valid := range strings.Split(v, "|") {
							ruleParts := strings.SplitN(valid, ":", 2)

							var k, v string
							if len(ruleParts) == 2 {
								k, v = ruleParts[0], ruleParts[1]
							} else {
								k = ruleParts[0]
							}

							ru := rule{Name: k, Value: v}

							switch k {
							case "in":
								values := strings.Split(v, ",")
								ru.Value1 = strings.Join(values, ", ")

								switch f.Type {
								case "string":
									for i, v := range values {
										values[i] = `"` + strings.Trim(v, `"`) + `"`
									}
								}

								ru.Value = values

							case "nullable":
								required = false
								continue F

							case "required":
								continue F
							}

							rules = append(rules, ru)
						}

						if required {
							rules = append([]rule{{Name: "required"}}, rules...)
						}

						var (
							tplField          = "m." + f.Name
							tplUnwrappedField = tplField
						)

						if strings.HasPrefix(f.Type, "*") {
							tplUnwrappedField = "orm.Unwrap(" + tplField + ")"
						}

						var validates bytes.Buffer
						for _, rule := range rules {
							var requireMap map[string]any
							if rule.Name == "required" {
								val, msg := checkValue(f.Type)
								requireMap = map[string]any{
									"Val": val,
									"Msg": msg,
								}
							}

							err := t.ExecuteTemplate(&validates, rule.Name, map[string]any{
								"Type":           f.Type,
								"Value":          rule.Value,
								"Value1":         rule.Value1,
								"Required":       requireMap,
								"Name":           f.Name,
								"Field":          tplField,
								"UnwrappedField": tplUnwrappedField,
							})

							if err != nil {
								log.Fatal(err)
							}
						}

						m.Validates = append(m.Validates, validates.String())

					case "rel":
						if rel := newRelation(f.Name, v, sf.Type, false); rel != nil {
							m.Relations[f.Name] = rel
							continue FL
						}

						log.Fatal("missing relation key")

					case "default":
						initial, _ := checkValue(f.Type)
						f.Default = &fieldDefault{Value: initialValue(f.Type, v), Initial: initial}

						if v == "now" || v == "updateNow" {
							f.Default.Value = "time.Now()"
							f.Default.Builtin = true
							if v == "updateNow" {
								f.Default.Trigger = "u"
							}
						}
					}
				}
			}
		}

		if f.Column == "" {
			f.Column = strings.ToLower(f.Name)
		}

		m.Fields = append(m.Fields, f)
	}

	if m.PK == nil {
		for _, f := range m.Fields {
			if strings.ToUpper(f.Name) == "ID" {
				m.PK = &modelPK{Name: "ID", Type: f.Type}
				break
			}
		}
	}

	if m.PK != nil {
		m.PK.Int = isNumPK(m.PK.Type)
		m.PK.Auto = m.PK.Auto && m.PK.Int
	}

	return m
}

func firstFieldName(l *ast.FieldList) string {
	if l.NumFields() > 0 {
		if id, ok := l.List[0].Type.(*ast.Ident); ok {
			return id.Name
		}
	}
	return ""
}

func typeName(m *model, e ast.Expr, isPtr bool) string {
	if e == nil {
		return ""
	}

	var sb strings.Builder
	printer.Fprint(&sb, fset, e)
	return sb.String()
}

func lowerFirst(s string) string {
	if len(s) > 0 {
		var chars []rune
		for i, c := range s {
			if i == 0 {
				chars = append(chars, unicode.ToLower(c))
			} else {
				chars = append(chars, c)
			}
		}
		return string(chars)
	}
	return s
}

func lowerField(s string) string {
	var chars []rune

	var count uint
	for _, c := range s {
		if unicode.IsUpper(c) {
			if count == 0 {
				chars = append(chars, unicode.ToLower(c))
				continue
			}
		} else {
			count = 1
		}

		chars = append(chars, c)
	}

	return string(chars)
}
