package main

import (
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	OPEN_DIR_ERROR = iota
	OPEN_FILE_ERROR
	OS_STAT_ERROR
	HASH_ERROR
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
	diff_content []string // get the modified file with content
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

	// copy contentdata in to the hasher
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
func ReadDir(dir_path string) *StatList {
	// init localStatList
	var localStatList = initStatList()

	// open the dir for read
	fs, err := os.ReadDir(dir_path)
	if err != nil {
		log.Fatal("Error Opening Dir ", err)
		os.Exit(OPEN_DIR_ERROR)
	}

	// check if file isReuglar
	// check if file isDir
	for _, f := range fs {
		fullpath := filepath.Join(dir_path, f.Name())

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


func callCommand() {
	cmd_clear := exec.Command("clear") 
	cmd_clear.Stdout = os.Stdout 
	cmd_clear.Stderr = os.Stderr 
	if err := cmd_clear.Run(); err != nil {
		log.Printf("Error <%s> \n", err ) 
	}

	absPath, err := filepath.Abs("../testdir")
	if err != nil {
		log.Fatal("error reading absolute path", err ) 
	}
	cout , err := exec.Command("go", "-C",absPath, "run", "main.go").CombinedOutput()
	if err != nil {
		log.Printf("Error CombinedOut <%s> \n", err ) 
	}
	fmt.Println(string(cout) )
}
func watchDir(dir_path string) {
	initStat := ReadDir(dir_path)
	fmt.Println("Data was initialized ...")
	fmt.Println("Starting watchDog  .....")
	for {
		time.Sleep(time.Second * 1)
		newStat := ReadDir(dir_path)
		if len(initStat.Map) < len(newStat.Map) {
			result := checkDiff(initStat, newStat)
			if len(result.diff_file) > 0 {
				log.Println("File or More was Added :")
				for _, fname := range result.diff_file {
					fmt.Printf("\t %s \n", fname)
				}
			}
			if len(result.diff_content) > 0 {
				log.Println("file was changed : ")
				for _, fname := range result.diff_content {
					fmt.Printf("\t %s \n", fname)
				}
			}
			callCommand()
		} else if len(initStat.Map) > len(newStat.Map) {
			result := checkDiff(newStat, initStat)
			if len(result.diff_file) > 0 {
				log.Println("File or More was Removed :")
				for _, fname := range result.diff_file {
					fmt.Printf("\t %s \n", fname)
				}
			}
			if len(result.diff_content) > 0 {
				log.Println("file was changed : ")
				for _, fname := range result.diff_content {
					fmt.Printf("\t %s \n", fname)
				}
			}
			callCommand()
		}else {
			result := checkDiff(newStat, initStat)
			if len(result.diff_file) > 0  || len(result.diff_content) > 0 {
			
				if len(result.diff_file) > 0 {
					log.Println("File or More was Removed :")
					for _, fname := range result.diff_file {
						fmt.Printf("\t %s \n", fname)
					}
				}
				if len(result.diff_content) > 0 {
					log.Println("file was changed : ")
					for _, fname := range result.diff_content {
						fmt.Printf("\t %s \n", fname)
					}
				}
				callCommand()
			}
		}

		initStat = newStat
	}
}
func main() {
	path_dir := "../testdir"
	watchDir(path_dir)
	fmt.Println("Hello World")
}
