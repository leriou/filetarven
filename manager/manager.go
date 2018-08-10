package manager

import (
	"crypto/md5"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"Sephiroth/utils"
	"gopkg.in/mgo.v2/bson"
	"gopkg.in/mgo.v2"
)

type FM struct {
	di   *utils.Di
	tool *utils.Tool
	db  *mgo.Collection
	total int
	hidden bool
}

const (
	DISPLAY_HIDDEN_FILE_DEFAULT = false
	TOTAL_DEFAULT = 0
	APPLIED_DEFAULT = false
)

func NewFm() *FM {
	fm := new(FM)
	fm.di = utils.NewDi()
	fm.tool = utils.NewTool()
	fm.db = fm.di.GetMongoDB().DB("local").C("files")
	fm.hidden = DISPLAY_HIDDEN_FILE_DEFAULT
	return fm
}

func (fm *FM) SetHidden(flag bool){
	fm.hidden = flag
}



func (fm *FM) Rename(old, new string) {
	os.Rename(old, new)
	fm.tool.Logging("INFO"," Rename "+ old +" to "+ new )
}

func (fm *FM) Remove(path string) {
	os.Remove(path)
	fm.tool.Logging("INFO"," Remove "+path)
}

/**
 * 扫描某文件夹的信息
 */
func (fm *FM) Scan(filepath string) {
	fm.total = TOTAL_DEFAULT
	// 将文件夹读入数据库 
	fm.SaveFileInfos(filepath)
	fm.tool.Logging("INFO", fmt.Sprintf(" Reading files success, total : %d ",fm.total))
}

func (fm *FM) SaveFileInfos(path string) {
	files := fm.ReadFileFromPath(path)
	fm.SaveFileInfo(files)
}

/**
 * 读入某路径下的所有文件
 */
 func (fm *FM) ReadFileFromPath(path string) []FileInfo{
	files := make([]FileInfo,0)
	err := filepath.Walk(path,
		func(path string, f os.FileInfo, err error) error {
			if f == nil {
				return err
			}
			if f.IsDir() {
				return nil
			}

			if (!fm.hidden && len(fm.tool.Regex("/\\.",path)) > 0 ) {
				return nil
			}
			fInfo := NewFileInfo()
			go (fm.GetFileMetaInfo(path, fInfo))
			files = append(files,*fInfo)
			return nil
		})
	if err != nil {
		fmt.Printf("filepath.Walk() returned %v \n", err)
	}
	return files
}

/**
 * 获取文件信息
 */
 func (fm *FM) GetFileMetaInfo(path string, finfo *FileInfo) bool {
	info, _ := os.Stat(path)
	data, _ := ioutil.ReadFile(path)
	finfo.Md5 = fmt.Sprintf("%x", md5.Sum(data))                 // md5
	finfo.ModTime = info.ModTime().Format("2006-01-02 15:04:05") // 修改时间
	finfo.IsDir = info.IsDir()                                   // 是否目录
	finfo.Mode = fmt.Sprintf("%s", info.Mode())                  // 文件权限
	finfo.Name = info.Name()                                     // 文件名
	finfo.Size = float64(info.Size())/1024                       // 文件大小
	finfo.Applied = APPLIED_DEFAULT
	finfo.Path = path
	finfo.NewPath = path
	return true
}


/**
 * 保存文件信息
 */
 func (fm *FM) SaveFileInfo(files []FileInfo){
	for _,file := range files{
		// 检查文件是否重复, 重复则更新,否则插入
		var fp []FileInfo
		condition := bson.M{"md5":file.Md5,"path":file.Path}
		fm.db.Find(condition).All(&fp)
		if len(fp) > 0 {
			for _,item := range fp {
				fmt.Println(item)
				// fm.db.Update(bson.M{"_id":fp._id},item)
			}
			
		} else {
			fm.db.Insert(file)
			fm.total++
			fm.tool.Logging("INFO", " file :"+file.Path+" done")
		}
	}
}

/**
 * 对某文件夹进行更新
 */
func (fm *FM) Apply(filepath string) {
	// 查找文件路径下的旧文件信息
	var files []FileInfo
	condition := bson.M{"path":bson.M{"$regex":filepath}}
	fm.db.Find(condition).All(&files)
	for _,a := range files{
		if a.Path != a.NewPath{
			fm.Rename(a.Path,a.NewPath)
		}
	}
	// 清空旧数据库文件信息
	fm.ClearPath(filepath)
	// 重新导入
	fm.Scan(filepath)
	//fm.ClearAll()
}

func (fm *FM) ClearPath(path string) {
	fm.db.RemoveAll(bson.M{"path":bson.M{"$regex":path}})
}
/**
 * 清理所有文件
 */
func (fm *FM) ClearAll() {
	fm.db.RemoveAll(bson.M{})
}
