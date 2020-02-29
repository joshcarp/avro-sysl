package main

import (
	"bytes"
	"github.com/anz-bank/sysl/pkg/sysl"
	printer "github.com/joshcarp/sysl-printer"
	pgs "github.com/lyft/protoc-gen-star"
	"github.com/sirupsen/logrus"
	"io"
	"regexp"
	"strings"
)

type PrinterModule struct {
	*pgs.ModuleBase
	pgs.Visitor
	prefix string
	w      io.Writer
	Log    *logrus.Logger
	Module *sysl.Module
}

const typeApplication = "Types"

var TypeMapping = map[string]sysl.Type_Primitive{"TYPE_BYTES": sysl.Type_BYTES, "TYPE_INT32": sysl.Type_INT, "TYPE_STRING": sysl.Type_STRING, "TYPE_BOOL": sysl.Type_BOOL}

func ASTPrinter() *PrinterModule { return &PrinterModule{ModuleBase: &pgs.ModuleBase{}} }

func (p *PrinterModule) Name() string { return "printer" }

func (p *PrinterModule) Execute(targets map[string]pgs.File, packages map[string]pgs.Package) []pgs.Artifact {
	buf := &bytes.Buffer{}
	if p.Log == nil {
		p.Log = logrus.New()
	}
	for _, f := range targets {

		p.Module = &sysl.Module{
		Apps: make(map[string]*sysl.Application, 0),
	}
	p.Module.Apps[typeApplication] = &sysl.Application{
		Name: &sysl.AppName{
		},
	}
	p.Module.Apps[typeApplication].Types = map[string]*sysl.Type{}
	fileName := strings.Replace(regexp.MustCompile(`(?m)\w*\.proto`).FindString(f.Name().String()), ".proto", "", -1)
	p.populateModule(f, buf)
	prin := printer.NewPrinter(buf)
	prin.PrintModule(p.Module)
	p.AddGeneratorFile(fileName+".sysl", buf.String())
	}


	return p.Artifacts()
}

func (p *PrinterModule) populateModule(f pgs.File, buf *bytes.Buffer) {
	p.Push(f.Name().String())
	defer p.Pop()

	buf.Reset()
	v := p.initPrintVisitor(buf, "")
	p.CheckErr(pgs.Walk(v, f), "unable to print AST tree")
	out := buf.String()
	if ok, _ := p.Parameters().Bool("log_tree"); ok {
		p.Logf("Proto Tree:\n%s", out)
	}

}

func (p *PrinterModule) initPrintVisitor(w io.Writer, prefix string) pgs.Visitor {
	p.prefix = prefix
	p.Visitor = pgs.PassThroughVisitor(p)
	p.w = w
	return p
}

func (v PrinterModule) VisitMessage(m pgs.Message) (pgs.Visitor, error) {
	attrDefs := make(map[string]*sysl.Type)

	var fieldName string
	var syslType *sysl.Type
	for _, e := range m.Fields() {
		fieldName, syslType = fieldToSysl(e)
		attrDefs[fieldName] = syslType
	}
	v.Module.Apps[typeApplication].Types[m.Name().String()] = &sysl.Type{
		Type: &sysl.Type_Tuple_{
			Tuple: &sysl.Type_Tuple{
				AttrDefs: attrDefs,
			},
		},
	}
	return v, nil
}

func (v PrinterModule) VisitService(s pgs.Service) (pgs.Visitor, error) {
	v.Module.Apps[s.Name().String()] = &sysl.Application{
		Name:      &sysl.AppName{Part: []string{s.Name().String()}},
		Endpoints: v.fillEndpoints(s.Methods()),
	}
	return nil, nil
}

func (v PrinterModule) fillEndpoints(methods []pgs.Method) map[string]*sysl.Endpoint {
	ep := make(map[string]*sysl.Endpoint, len(methods))
	for _, method := range methods {
		calls := customOption(method)
		syslCalls := []*sysl.Statement{}
		for _, call := range calls {
			syslCalls = append(syslCalls, &sysl.Statement{
				Stmt: &sysl.Statement_Call{
					Call: &sysl.Call{
						Target: &sysl.AppName{
							Part: []string{call.Service},
						},
						Endpoint: call.Method,
					},
				},
			},
			)
		}
		ep[method.Name().String()] = &sysl.Endpoint{
			Name:     method.Name().String(),
			LongName: method.FullyQualifiedName(),
			Param: []*sysl.Param{{
				Name: "input",
				Type: messageToSysl(method.Input()),
			}},
			Stmt: append(syslCalls, &sysl.Statement{Stmt: &sysl.Statement_Ret{Ret: &sysl.Return{
				Payload: method.Output().Name().String(),
			}}}),
		}

	}
	return ep
}

func (v PrinterModule) VisitEnumValue(ev pgs.EnumValue) (pgs.Visitor, error) {
	return nil, nil
}

func (v PrinterModule) VisitField(f pgs.Field) (pgs.Visitor, error) {
	return nil, nil
}

func (v PrinterModule) VisitMethod(m pgs.Method) (pgs.Visitor, error) {
	return nil, nil
}

func (v PrinterModule) VisitFile(f pgs.File) (pgs.Visitor, error) {
	return v, nil
}

func (v PrinterModule) VisitEnum(e pgs.Enum) (pgs.Visitor, error) {
	return nil, nil
}
