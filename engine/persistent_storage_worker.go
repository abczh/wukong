package engine

import (
	"bytes"
	"encoding/gob"
	"github.com/henrylee2cn/wukong/types"
	"io"
	"log"
	"sync/atomic"
)

type persistentStorageIndexDocumentRequest struct {
	docId string
	data  types.DocumentIndexData
}

func (engine *Engine) persistentStorageIndexDocumentWorker(shard int) {
	for {
		request := <-engine.persistentStorageIndexDocumentChannels[shard]

		// 得到key
		b := []byte(request.docId)

		// 得到value
		var buf bytes.Buffer
		enc := gob.NewEncoder(&buf)
		err := enc.Encode(request.data)
		if err != nil {
			atomic.AddUint64(&engine.numDocumentsStored, 1)
			continue
		}

		// 将key-value写入数据库
		engine.dbs[shard].Set(b, buf.Bytes())
		atomic.AddUint64(&engine.numDocumentsStored, 1)
	}
}

func (engine *Engine) persistentStorageRemoveDocumentWorker(docId string, shard uint32) {
	// 得到key
	b := []byte(docId)

	// 从数据库删除该key
	engine.dbs[shard].Delete(b)
}

func (engine *Engine) persistentStorageInitWorker(shard int) {
	iter, err := engine.dbs[shard].SeekFirst()
	if err == io.EOF {
		engine.persistentStorageInitChannel <- true
		return
	} else if err != nil {
		engine.persistentStorageInitChannel <- true
		log.Fatal("无法遍历数据库")
	}

	for {
		key, value, err := iter.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			continue
		}

		// 得到docID
		docId := string(key)

		// 得到data
		buf := bytes.NewReader(value)
		dec := gob.NewDecoder(buf)
		var data types.DocumentIndexData
		err = dec.Decode(&data)
		if err != nil {
			continue
		}

		// 添加索引
		engine.internalIndexDocument(docId, data)
	}
	engine.persistentStorageInitChannel <- true
}
