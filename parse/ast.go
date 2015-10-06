package parse

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"

	"github.com/sdboyer/memoize/gen"
)

type Constraint uint8

const (
	NeedsKeyFunc Constraint = 1 << iota
	Inlinable
)

// A FileSet represents all of the is the in-memory representation of a
// parsed file.
type FileSet struct {
	Package    string                   // package name
	Funcs      map[string]*ast.FuncDecl // func decls in file
	Identities map[string]gen.Elem      // processed from specs
	Directives []string                 // raw preprocessor directives
}

func File(name string) (*FileSet, error) {
	//pushstate(name)
	//defer popstate()
	fs := &FileSet{
		Funcs:      make(map[string]*ast.FuncDecl),
		Identities: make(map[string]gen.Elem),
	}

	fset := token.NewFileSet()
	finfo, err := os.Stat(name)
	if err != nil {
		return nil, err
	}
	if finfo.IsDir() {
		// parse everything in in the specified dir
		pkgs, err := parser.ParseDir(fset, name, nil, parser.ParseComments)
		if err != nil {
			return nil, err
		}
		if len(pkgs) != 1 {
			return nil, fmt.Errorf("multiple packages in directory: %s", name)
		}
		// sdb: plucks out the first package, which'll be the main lib and not a test pkg or anything
		var one *ast.Package
		for _, nm := range pkgs {
			one = nm
			break
		}
		fs.Package = one.Name
		for _, fl := range one.Files {
			//pushstate(fl.Name.Name)
			fs.Directives = append(fs.Directives, yieldComments(fl.Comments)...)
			fs.getFuncDecls(fl)
			//popstate()
		}
	} else {
		f, err := parser.ParseFile(fset, name, nil, parser.ParseComments)
		if err != nil {
			return nil, err
		}
		fs.Package = f.Name.Name
		fs.Directives = yieldComments(f.Comments)
		fs.getFuncDecls(f)
	}

	if len(fs.Funcs) == 0 {
		return nil, fmt.Errorf("no function or method declarations to memoize in %s", name)
	}

	fs.process()
	fs.applyDirectives()
	fs.propInline()

	return fs, nil
}

// getTypeSpecs extracts all of the *ast.TypeSpecs in the file
// into fs.Identities, but does not set the actual element
func (fs *FileSet) getFuncDecls(f *ast.File) {
	// check all declarations...
	for i := range f.Decls {
		// for FuncDecls, and record them into the fileset
		if fd, ok := f.Decls[i].(*ast.FuncDecl); ok {
			fs.Funcs[fd.Name.Name] = fd
		}
	}
}

// process populates f.Identities from the contents of f.Funcs
func (f *FileSet) process() {

	deferred := make(linkset)
parse:
	for name, def := range f.Specs {
		pushstate(name)
		el := f.parseExpr(def)
		if el == nil {
			warnln("failed to parse")
			popstate()
			continue parse
		}
		// push unresolved identities into
		// the graph of links and resolve after
		// we've handled every possible named type.
		if be, ok := el.(*gen.BaseElem); ok && be.Value == gen.IDENT {
			deferred[name] = be
			popstate()
			continue parse
		}
		el.Alias(name)
		f.Identities[name] = el
		popstate()
	}

	if len(deferred) > 0 {
		f.resolve(deferred)
	}
}
