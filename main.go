package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/jckuester/terratools/gen"

	"github.com/apex/log"
	"github.com/apex/log/handlers/cli"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/private/util"
)

const codeLayout = `// Code is generated by terratools. DO NOT EDIT.

%s
package %s

%s
`

func main() {
	log.SetHandler(cli.Default)
	log.SetLevel(log.DebugLevel)

	/*
		resourceTypes := ResourceTypes()

		err := writeResourceTypes("gen", resourceTypes)
		if err != nil {
			log.Fatal(err.Error())
		}

		log.Infof("Generated list of Terraform AWS resource types: %d", len(resourceTypes))

		resourceFileNames := map[string]string{}
		for _, rType := range resourceTypes {
			fileName, err := GetResourceFileName(rType)
			if err != nil {
				log.Fatal(err.Error())
			}
			resourceFileNames[rType] = fileName
		}

		err = writeResourceFileNames("gen", resourceFileNames)
		if err != nil {
			log.Fatal(err.Error())
		}

		log.Infof("Generated map of resource type -> name of file that implements resource: %d", len(resourceFileNames))

		resourceService := map[string]string{}
		for _, rType := range resourceTypes {
			rFileName, ok := resourceFileNames[rType]
			if !ok {
				log.Fatal("cannot find filename for resource type")
			}

			service, err := ResourceService(rType, rFileName)
			if err != nil {
				log.WithError(err).WithField("resource", rType).Warn("could not identify AWS service for resource")
				continue
			}
			resourceService[rType] = service
		}

		err = writeResourceServices("gen", resourceService)
		if err != nil {
			log.Fatal(err.Error())
		}

		log.Infof("Generated map of resource type -> AWS service: %d/%d", len(resourceService), len(ResourceTypes()))
	*/
	resourceIDs := map[string]string{}
	for rType, fileName := range gen.ResourceFileNames {
		resourceID, err := GetResourceID(fileName)
		if err != nil {
			continue
		}
		resourceIDs[rType] = resourceID
	}

	err := writeResourceIDs("gen", resourceIDs)
	if err != nil {
		log.Fatal(err.Error())
	}

	log.Infof("Generated map of resource type -> AWS API struct field name of ID: %d", len(resourceIDs))
}

func writeResourceTypes(outputPath string, resourceTypes []string) error {
	err := os.MkdirAll(outputPath, 0775)
	if err != nil {
		return fmt.Errorf("failed to create directory: %s", err)
	}

	err = writeGoFile(
		filepath.Join(outputPath, "resourceTypes.go"),
		codeLayout,
		"",
		"gen",
		ResourceTypesGoCode(resourceTypes),
	)
	if err != nil {
		return fmt.Errorf("failed to write list of resource types to file: %s", err)
	}

	return nil
}

func writeResourceFileNames(outputPath string, resourceFileNames map[string]string) error {
	err := os.MkdirAll(outputPath, 0775)
	if err != nil {
		return fmt.Errorf("failed to create directory: %s", err)
	}

	err = writeGoFile(
		filepath.Join(outputPath, "resourceFileNames.go"),
		codeLayout,
		"",
		"gen",
		ResourceFileNamesGoCode(resourceFileNames),
	)
	if err != nil {
		return fmt.Errorf("failed to write map of resource file names to file: %s", err)
	}

	return nil
}

func writeResourceIDs(outputPath string, resourceIDs map[string]string) error {
	err := os.MkdirAll(outputPath, 0775)
	if err != nil {
		return fmt.Errorf("failed to create directory: %s", err)
	}

	err = writeGoFile(
		filepath.Join(outputPath, "resourceIDs.go"),
		codeLayout,
		"",
		"gen",
		ResourceIDsGoCode(resourceIDs),
	)
	if err != nil {
		return fmt.Errorf("failed to write map of resource IDs to file: %s", err)
	}

	return nil
}

// ResourceTypesGoCode generates the Go code for the list of Terraform resource types.
func ResourceTypesGoCode(terraformTypes []string) string {
	var buf bytes.Buffer
	err := resourceTypesTmpl.Execute(&buf, terraformTypes)
	if err != nil {
		panic(err)
	}

	return strings.TrimSpace(buf.String())
}

// ResourceFileNamesGoCode generates the Go code for the map of Terraform resource file names.
func ResourceFileNamesGoCode(resourceFileNames map[string]string) string {
	var buf bytes.Buffer
	err := resourceFileNamesTmpl.Execute(&buf, resourceFileNames)
	if err != nil {
		panic(err)
	}

	return strings.TrimSpace(buf.String())

}

// ResourceIDsGoCode generates the Go code for the map of Terraform resource IDs.
func ResourceIDsGoCode(resourceIDs map[string]string) string {
	var buf bytes.Buffer
	err := resourceIDsTmpl.Execute(&buf, resourceIDs)
	if err != nil {
		panic(err)
	}

	return strings.TrimSpace(buf.String())

}

var resourceTypesTmpl = template.Must(template.New("resourceTypes").Parse(`
// ResourceTypes is a list of all resource types supported by the Terraform AWS Provider.
var ResourceTypes = []string{
{{range .}}"{{.}}",
{{end}}}
`))

var resourceFileNamesTmpl = template.Must(template.New("resourceFileNames").Parse(`
// ResourceFileNames stores the name of the file that implements the resource type in the Terraform AWS Provider.
var ResourceFileNames = map[string]string{
{{range $key, $value := .}}"{{$key}}": "{{$value}}",
{{end}}}
`))

var resourceIDsTmpl = template.Must(template.New("resourceIDs").Parse(`
// ResourceIDs stores the name of the struct field of the AWS API used as ID for each Terraform resource type.
var ResourceIDs = map[string]string{
{{range $key, $value := .}}"{{$key}}": "{{$value}}",
{{end}}}
`))

func writeGoFile(file string, layout string, args ...interface{}) error {
	return ioutil.WriteFile(file, []byte(util.GoFmt(fmt.Sprintf(layout, args...))), 0664)
}

// ResourceTypes returns a list of all resource types supported by the Terraform AWS Provider.
func ResourceTypes() []string {
	node, err := parser.ParseFile(token.NewFileSet(),
		"/home/jan/git/github.com/yoyolabsio/terraform-provider-aws/aws/provider.go",
		nil, 0)
	if err != nil {
		log.Fatalf(err.Error())
	}

	var result []string

	ast.Inspect(node, func(n ast.Node) bool {
		// look for a map called "ResourcesMap" that contains all the resource types
		resourcesMap, ok := n.(*ast.KeyValueExpr)
		if !ok {
			return true
		}

		resourceMapName, ok := resourcesMap.Key.(*ast.Ident)
		if !ok {
			return false
		}

		if resourceMapName.Name != "ResourcesMap" {
			return false
		}

		ast.Inspect(resourcesMap, func(n ast.Node) bool {
			mapEntry, ok := n.(*ast.KeyValueExpr)
			if !ok {
				return true
			}

			resourceType, ok := mapEntry.Key.(*ast.BasicLit)
			if !ok {
				return true
			}

			result = append(result, strings.Trim(resourceType.Value, "\""))

			return true
		})

		return true
	})

	return result
}

// GetResourceFileName returns the name of the file that implements the resource of the given resource type.
func GetResourceFileName(resourceType string) (string, error) {
	node, err := parser.ParseFile(token.NewFileSet(),
		"/home/jan/git/github.com/yoyolabsio/terraform-provider-aws/aws/provider.go",
		nil, 0)
	if err != nil {
		return "", err
	}

	var result *string

	ast.Inspect(node, func(n ast.Node) bool {
		mapEntry, ok := n.(*ast.KeyValueExpr)
		if !ok {
			return true
		}

		key, ok := mapEntry.Key.(*ast.BasicLit)
		if !ok {
			return true
		}

		if resourceType != strings.Trim(key.Value, "\"") {
			return false
		}

		resource, ok := mapEntry.Value.(*ast.CallExpr)
		if !ok {
			return false
		}

		resourceFuncName, ok := resource.Fun.(*ast.Ident)
		if !ok {
			return false
		}

		files, err := ioutil.ReadDir("/home/jan/git/github.com/yoyolabsio/terraform-provider-aws/aws")
		if err != nil {
			log.Fatal(err.Error())
		}

		for _, f := range files {
			if strings.HasPrefix(f.Name(), "resource_aws") {
				node, err := parser.ParseFile(token.NewFileSet(),
					"/home/jan/git/github.com/yoyolabsio/terraform-provider-aws/aws/"+f.Name(),
					nil, 0)
				if err != nil {
					log.Fatal(err.Error())
				}

				ast.Inspect(node, func(n ast.Node) bool {
					fn, ok := n.(*ast.FuncDecl)
					if !ok {
						return true
					}

					if fn.Name.Name == resourceFuncName.Name {
						result = aws.String(f.Name())
						return false
					}

					return true
				})
			}
		}
		return true
	})

	if result != nil {
		return *result, nil
	}

	return "", fmt.Errorf("no file found that implements resource type")
}

func GetResourceID(fileName string) (string, error) {
	fset := token.NewFileSet()

	node, err := parser.ParseFile(fset,
		"/home/jan/git/github.com/yoyolabsio/terraform-provider-aws/aws/"+fileName,
		nil, 0)
	if err != nil {
		return "", err
	}

	var result *string

	ast.Inspect(node, func(n ast.Node) bool {
		// find d.SetId(...) function calls
		fn, ok := n.(*ast.CallExpr)
		if ok {
			if fun, ok := fn.Fun.(*ast.SelectorExpr); ok {
				funcName := fun.Sel.Name
				if funcName == "SetId" {
					// setId has only one argument
					switch x := fn.Args[0].(type) {
					case *ast.BasicLit:
						// ignore d.SetId("")
					case *ast.Ident:
						if strings.Contains(strings.ToLower(x.Name), "name") {
							// if the identifier contains "name", we know it's the
							// name identifying a resource but figure out the real AWS struct field name
							// at a later stage
							result = aws.String("NAME_PLACEHOLDER")
						}

						// handle the following kind of expressions:
						//   id := *res.ImageId
						//   d.SetId(id)
						ast.Inspect(node, func(n ast.Node) bool {
							ass, ok := n.(*ast.AssignStmt)
							if !ok {
								return true
							}

							if ass.Tok != token.DEFINE {
								return true
							}

							leftAss, ok := ass.Lhs[0].(ast.Expr)
							if !ok {
								return true
							}

							ident, ok := leftAss.(*ast.Ident)
							if !ok {
								return true
							}

							if ident.Name != x.Name {
								return true
							}

							rightAss, ok := ass.Rhs[0].(ast.Expr)
							if !ok {
								return true
							}

							switch x := rightAss.(type) {
							case *ast.StarExpr:
								identRight, ok := x.X.(*ast.SelectorExpr)
								if !ok {
									return true
								}

								result = &identRight.Sel.Name
							}

							return true
						})
					case *ast.StarExpr:
						// handle the following kind of expressions: d.SetId(*vpc.VpcId)
						ident, ok := x.X.(*ast.SelectorExpr)
						if !ok {
							return true
						}

						result = &ident.Sel.Name
					case *ast.CallExpr:
						// handle the following kind of expressions:
						//	d.SetId(aws.StringValue(output.CertificateAuthorityArn))
						fn, ok := x.Fun.(*ast.SelectorExpr)
						if !ok {
							return true
						}

						if fn.Sel.Name != "StringValue" {
							return true
						}

						// handle the following kind of expressions: d.SetId(*vpc.VpcId)
						ident, ok := x.Args[0].(*ast.SelectorExpr)
						if !ok {
							return true
						}

						result = &ident.Sel.Name
					}
				}
			}
		}

		return true
	})

	if result != nil {
		return *result, nil
	}

	return "", fmt.Errorf("no ID found for resource type")
}
