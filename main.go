package main

import (
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"flag"
	"path/filepath"
	"path"
	"strings"
	"time"
)

const (
	RED = "\033[31m" 	// deleted file 
	GREEN="\033[32m" 	// file's content changed 
	MAGENTA="\033[035m" 	// file's name changed 
	YELLOW ="\033[033m" 	// new file
	RESET="\033[0m"  	// reset color 

	OPEN_DIR_ERROR = iota
	OPEN_FILE_ERROR
	OS_STAT_ERROR
	HASH_ERROR
	ARGS_ERROR 
)
type FileStat struct {
	Name        string    // name by fs.Name()
	Size        int64     // size by fs.Size()
	Modified_at time.Time // time of modifcation by fs.TimeMod
	hashContent string    // content hashed by HashContent
}

type StatList struct {
	Map map[string]*FileStat
}
type Result struct {
	diff_file    []string // get the deleted or newest file in a dir
	diff_content []string // get the modified file
}

// initialise the Statlist to prevent error with nil pointer(map)
func initStatList() *StatList {
	return &StatList{
		Map: make(map[string]*FileStat),
	}
}

// hash the content of a file using sha256
// return result as []byte (maybe string )
func HashContent(filename string) string {
	// open file
	//
	file, err := os.Open(filename)
	if err != nil {
		log.Fatal("error opening file ", err)
		os.Exit(OPEN_FILE_ERROR)
	}
	// close file
	defer file.Close()

	// init hasher with hash.New()
	hasher := sha256.New()

	// copy content data into the hasher
	if _, err := io.Copy(hasher, file); err != nil {
		log.Fatal("error hashing content of the file ", err)
		os.Exit(HASH_ERROR)
	}

	hash_content := fmt.Sprintf(" %x ", hasher.Sum(nil))
	// return the sum(nil)
	return hash_content

}

// get info of the file ( size , name , modication_time , content_modification) using stat
// using hash_content functinon for the hasContent property
func GetFileStat(filename string) *FileStat {
	hash_content := HashContent(filename)
	fs, err := os.Stat(filename)
	if err != nil {
		log.Fatal("error occred os.Stat ", err)
		os.Exit(OS_STAT_ERROR)
	}
	return &FileStat{
		Name:        filename,
		Size:        fs.Size(),
		Modified_at: fs.ModTime(),
		hashContent: hash_content,
	}

}
func ReadDir(absPath string) *StatList {
	// init localStatList
	var localStatList = initStatList()

	// open the dir for read
	fs, err := os.ReadDir(absPath)
	if err != nil {
		log.Fatal("Error Opening Dir ", err)
		os.Exit(OPEN_DIR_ERROR)
	}

	// check if file isReuglar(todo)
	// check if file isDir
	for _, f := range fs {
		fullpath := filepath.Join(absPath, f.Name())
		// escape hidden dir as (.git| .config)
		if string(f.Name()[0]) == "." {
			continue
		}
		if f.IsDir() {
			newStatList := ReadDir(fullpath)
			for key, data := range newStatList.Map {
				localStatList.Map[key] = data
			}
		} else {
			localStatList.Map[fullpath] = GetFileStat(fullpath)
		}
	}
	return localStatList
}
func checkDiff(initStat, newStat *StatList) *Result {
	result := &Result{}
	// assert len(initStat.Map ) < len(newStat.Map)
	for key, newData := range newStat.Map {
		if oldData, ok := initStat.Map[key]; !ok {
			result.diff_file = append(result.diff_file, key)
		} else {
			// if file has different fs.ModTime()
			// must check the hashcontent if it was modified
			// otherwise there is no changed occured to the file
			if time.Time.Compare(oldData.Modified_at, newData.Modified_at) != 0 {
				if strings.Compare(oldData.hashContent, newData.hashContent) != 0 {
					result.diff_content = append(result.diff_content, key)

				}
			}
		}
	}
	return result

}

func clearScreen() {
	cmd_clear := exec.Command("clear") 
	cmd_clear.Stdout = os.Stdout 
	cmd_clear.Stderr = os.Stderr 
	if err := cmd_clear.Run(); err != nil {
		log.Printf("Error <%s> \n", err ) 
	}
}
func callCommand(absPath, target string ) {
	comp := getCompiler(target) 
	cmd := exec.Command(comp[0] , comp[1],target)
	cmd.Dir = absPath
	cout , err := cmd.CombinedOutput()
	if err != nil {
		log.Printf(MAGENTA + "Error Cmd.CombinedOutput : %s\n" + RESET, err ) 
		log.Printf(RED + "%s\n" + RESET, cout ) 
	}else {

		// handle outputs
		// seperate & colirify success output , error output 
		fmt.Println(cmd.Dir)
		fmt.Printf("\t\t_GAZE_OUTPUT_\n\n"+ GREEN + "%s" + RESET, string(cout) )
	}
}

// not universal, but works fine for now 
// using build is more efficient i guess 
func getCompiler(target string ) []string {
	switch path.Ext(target) {
	case ".go":
		return []string{"go", "run"}
	case ".rs" :
		return []string{"cargo", "run"}
	}
	return []string{"/bin/sh","-c"}
}


// parsing path (return not absolute_path) , and target file 
func parseArgs()(string, string) {
	var dir string 
	var target string 
	flag.StringVar(&dir, "dir", ".", "the directory required to run the program")
	flag.StringVar(&target, "tf", "main.go", "terget file to be executed in the dir recommanded")
	flag.Parse()
	if len(os.Args) < 5 || len(os.Args) > 5  {
	
		log.Printf("USAGE <%s> -dir <dirpath> -tf <target_file> \n", os.Args[0])
		os.Exit(ARGS_ERROR)
	}
	fmt.Println(dir, target) 
	return dir, target 
}
func (result *Result)HandleChanges(message ,color  string) {
	clearScreen()
	if len(result.diff_file) > 0 {
    		log.Println(message)
    		for _, fname := range result.diff_file{
        		fmt.Printf(color  + "\t<%s> "+RESET+"\n", fname)
    		}
	}
	if len(result.diff_content) >0 {
    		log.Println("File or More was changed")
    		for _, fname := range result.diff_content {
        		fmt.Printf(GREEN + "\t<%s>"+RESET+"\n", fname)
    		}
	}
}
func watchDir() {
	dir, target := parseArgs()
	absPath, err := filepath.Abs(dir) 
	if err != nil {
		log.Fatalf("error no dir <%s>, <%s> \n", err )
	}
	fmt.Printf("checking dir : %s with target : %s \n", absPath, target)
	callCommand(absPath, target) 
	initStat := ReadDir(absPath)
	fmt.Println("Data was initialized ...")
	fmt.Println("Starting watchDog  .....")
	for {
		time.Sleep(time.Second * 1)
		newStat := ReadDir(absPath)
		if len(initStat.Map) < len(newStat.Map) {
			result := checkDiff(initStat, newStat)
			result.HandleChanges("File or More was Added :",YELLOW) 
			callCommand(absPath, target)
		} else if len(initStat.Map) > len(newStat.Map) {
			result := checkDiff(newStat, initStat)
			result.HandleChanges("File or More was Removed", RED)
			callCommand(absPath, target)
		}else {
			result := checkDiff(newStat, initStat)
			if len(result.diff_file) > 0  || len(result.diff_content) > 0 {
				result.HandleChanges("File or More was renamed or mouved or copied", MAGENTA)			
				callCommand(absPath, target)
			}
		}

		initStat = newStat
	}
}
func main() {
	watchDir()
}
