package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type Misskey struct {
	Site  string
	Token string
}

type emoji struct {
	FileID      string
	FileName    string
	IsSensitive bool
}

var localOnly bool

func main() {
	var mi = Misskey{
		Site:  os.Getenv("MISSKEY_SITE"),
		Token: os.Getenv("MISSKEY_TOKEN"),
	}
	if mi.Site == "" || mi.Token == "" {
		panic("请设置环境变量 MISSKEY_SITE 和 MISSKEY_TOKEN")
	}
	var folderId string
	print("请输入 emoji 文件夹的 ID: ")
	fmt.Scanln(&folderId)
	var category string
	print("请输入 emoji 的分类: ")
	// 需要将空格也读入
	reader := bufio.NewReader(os.Stdin)
	category, _ = reader.ReadString('\n')
	// 去除末尾的换行符
	category = category[:len(category)-1]
	for {
		var input string
		print("是否只在本地使用(localOnly)？(y/n): ")
		fmt.Scanln(&input)
		if input == "y" || input == "Y" {
			localOnly = true
			break
		} else if input == "n" || input == "N" {
			localOnly = false
			break
		} else {
			println("请输入 y 或 n")
		}
	}

	// 获取 emoji 文件夹下的 emoji
	var emojis, err = mi.getFileIds(folderId)
	if err != nil {
		panic(err)
	}
	// 添加 emoji
	for _, v := range emojis {
		err = mi.addEmoji(v, category)
		if err != nil {
			println(err.Error())
		} else {
			println(v.FileName + " 添加成功")
		}
	}
	println("添加完成，按回车键退出")
	fmt.Scanln()
}

// 添加 emoji
func (mi *Misskey) addEmoji(emoji emoji, category string) error {
	type requestStruct struct {
		Token                                   string        `json:"i"`
		Name                                    string        `json:"name"`
		FileID                                  string        `json:"fileId"`
		Category                                string        `json:"category"`
		Aliases                                 []string      `json:"aliases"`
		License                                 interface{}   `json:"license"`
		IsSensitive                             bool          `json:"isSensitive"`
		LocalOnly                               bool          `json:"localOnly"`
		RoleIdsThatCanBeUsedThisEmojiAsReaction []interface{} `json:"roleIdsThatCanBeUsedThisEmojiAsReaction"`
	}
	var req = requestStruct{
		Token: mi.Token,
		// 去除文件后缀
		Name:                                    emoji.FileName[:len(emoji.FileName)-len(filepath.Ext(emoji.FileName))],
		FileID:                                  emoji.FileID,
		Category:                                category,
		Aliases:                                 []string{},
		License:                                 nil,
		IsSensitive:                             emoji.IsSensitive,
		LocalOnly:                               localOnly,
		RoleIdsThatCanBeUsedThisEmojiAsReaction: []interface{}{},
	}
	var dataBytes, _ = json.Marshal(req)
	resp, err := http.Post(mi.Site+"/api/admin/emoji/add", "application/json", bytes.NewBuffer(dataBytes))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 204 {
		// 解析错误响应到结构体
		type respError struct {
			Error struct {
				Message string `json:"message"`
				Code    string `json:"code"`
				ID      string `json:"id"`
			} `json:"error"`
		}
		var respData respError
		err = json.NewDecoder(resp.Body).Decode(&respData)
		if err != nil {
			return fmt.Errorf("%s add failed: %s", emoji.FileName, respData.Error.Message)
		}
	}
	return nil
}

// 获取文件夹下的文件 ID 列表
func (mi *Misskey) getFileIds(foldId string) ([]emoji, error) {
	type request struct {
		Token    string      `json:"i"`
		Limit    int         `json:"limit"`
		FolderID interface{} `json:"folderId"`
		Sort     string      `json:"sort"`
	}
	type respStruct []struct {
		ID           string    `json:"id"`
		CreatedAt    time.Time `json:"createdAt"`
		Name         string    `json:"name"`
		Type         string    `json:"type"`
		Md5          string    `json:"md5"`
		IsSensitive  bool      `json:"isSensitive"`
		URL          string    `json:"url"`
		ThumbnailURL string    `json:"thumbnailUrl"`
		FolderID     string    `json:"folderId"`
	}
	var req = request{
		Token:    mi.Token,
		Limit:    100,
		FolderID: foldId,
		Sort:     "+name",
	}
	// 发送请求
	var dataBytes, _ = json.Marshal(req)
	resp, err := http.Post(mi.Site+"/api/drive/files", "application/json", bytes.NewBuffer(dataBytes))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	// 解析响应
	var respData respStruct
	err = json.NewDecoder(resp.Body).Decode(&respData)
	if err != nil {
		return nil, err
	}
	var emojis []emoji
	for _, v := range respData {
		emojis = append(emojis, emoji{
			FileID:      v.ID,
			FileName:    v.Name,
			IsSensitive: v.IsSensitive,
		})
	}
	return emojis, nil
}
