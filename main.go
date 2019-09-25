package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

var (
	pkgName     string
	pkgPath     string
	excludePath string
	dotFile     string
	depth       int
)

func depthParse(s, tag string) string {
	lens := len(tag)
	src := s[lens:]
	if len(src) != 0 && src[0] == '/' {
		src = src[1:]
	}
	parts := strings.Split(src, "/")

	pkgs := strings.Split(pkgName, "/")
	ret := pkgs[len(pkgs)-1]
	for i := 0; i < depth && i < len(parts); i++ {
		if strings.Contains(parts[i], ".go") {
			break
		}
		ret += "/" + parts[i]
	}
	//fmt.Println(ret)
	return ret
}

func analyseGoFile(path string) error {
	dst := depthParse(path, pkgPath)

	srcMap := make(map[string]bool)

	f, err := os.Open(path)
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		return err
	}
	defer f.Close()

	br := bufio.NewReader(f)
	flag := false
	for {
		a, _, err := br.ReadLine()
		if err == io.EOF {
			break
		}
		line := strings.Trim(string(a), " ")
		if line == "import (" {
			flag = true
		}
		if line == ")" {
			flag = false
		}
		if flag == true {
			if indx := strings.Index(line, pkgName); indx != -1 {
				srcPkg := depthParse(strings.Trim(line[indx:], "\""), pkgName)

				if _, ok := srcMap[srcPkg]; !ok {
					//fmt.Println("ok pkg", srcPkg)
					srcMap[srcPkg] = true
				}
			}
		}
	}

	if excludePath != "" {
		for _, exclude := range strings.Split(excludePath, ",") {
			if strings.Contains(path, exclude) {
				//fmt.Println("drop src ", dst)
				return nil
			}
		}
	}
	fd, _ := os.OpenFile(dotFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	defer fd.Close()
	for k, _ := range srcMap {
		if excludePath != "" {
			var b bool
			for _, exclude := range strings.Split(excludePath, ",") {
				if strings.Contains(k, exclude) {
					//fmt.Println("drop dst", k, exclude)
					b = true
					break
				}
			}
			if b {
				continue
			}
		}
		fd.Write([]byte(fmt.Sprintf("\t\"%s\" -> \"%s\"\n", dst, k)))

	}
	return nil
}

func analyseDir(dir string) error {
	fd, _ := os.OpenFile(dotFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	fd.Write([]byte("digraph G {\n"))
	fd.Close()
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Printf("prevent panic by handling failure accessing a path %q: %v\n", dir, err)
			return err
		}

		if !info.IsDir() {
			if strings.Contains(path, ".go") && !strings.Contains(path, pkgPath+"/vendor") {
				if !strings.Contains(path, "_test.go") {
					//fmt.Printf("visited file: %q\n", path)
					analyseGoFile(path)
				}

			}

		}
		return nil
	})

	if err != nil {
		return err
	}

	fd, _ = os.OpenFile(dotFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	fd.Write([]byte("}\n"))
	fd.Close()
	return nil
}

func processDotFile() {
	fileContent := make(map[string]bool)
	f, err := os.Open(dotFile)
	if err != nil {
		fmt.Printf("Error: %s\n", err)
	}

	br := bufio.NewReader(f)

	for {
		a, _, err := br.ReadLine()
		if err == io.EOF {
			break
		}

		if _, ok := fileContent[string(a)]; !ok {
			fileContent[string(a)] = true
		}
	}
	f.Close()

	content := "digraph G {\n"
	for k, _ := range fileContent {
		if k == "digraph G {" || k == "}" {
			continue
		}
		content += k + "\n"
	}
	content += "}\n"
	ioutil.WriteFile(dotFile, []byte(content), 0644)
}

func main() {
	flag.StringVar(&pkgName, "pkg_name", "", "the package name")
	flag.StringVar(&excludePath, "exclude_path", "", "the exclude package path")
	flag.StringVar(&pkgPath, "pkg_path", "", "the package path")
	flag.StringVar(&dotFile, "dot_file_path", "godag.dot", "we generated the .dot file path")
	flag.IntVar(&depth, "depth", 1, "dependencies analyse depth")
	flag.Parse()

	if len(pkgPath) == 0 || len(pkgName) == 0 {
		fmt.Println("You MUST assign the package name and path")
		return
	}

	fmt.Println("pkgName:", pkgName)
	fmt.Println("pkgPath:", pkgPath)
	fmt.Println("dotFile:", dotFile)
	fmt.Println("excludePath:", excludePath)

	analyseDir(pkgPath)
	processDotFile()
}
