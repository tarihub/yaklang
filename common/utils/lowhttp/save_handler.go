package lowhttp

import (
	"sync"

	"github.com/google/uuid"
	"github.com/yaklang/yaklang/common/log"
	"github.com/yaklang/yaklang/common/utils"
)

type saveHTTPFlowHandler func(https bool, req []byte, rsp []byte, url string, remoteAddr string, reqSource string, runtimeId string, fromPlugin string, hiddenIndex string, payloads []string)

var saveHTTPFlowFunc saveHTTPFlowHandler

func RegisterSaveHTTPFlowHandler(h saveHTTPFlowHandler) {
	m := new(sync.Mutex)
	saveHTTPFlowFunc = func(https bool, req []byte, rsp []byte, url string, remoteAddr string, reqSource string, runtimeId string, fromPlugin string, hiddenIndex string, payloads []string) {
		m.Lock()
		defer m.Unlock()

		defer func() {
			if err := recover(); err != nil {
				log.Errorf("call lowhttp.saveHTTPFlowFunc panic: %s", err)
			}
		}()
		h(https, req, rsp, url, remoteAddr, reqSource, runtimeId, fromPlugin, hiddenIndex, payloads)
	}
}

func SaveResponse(r *LowhttpResponse) {
	if saveHTTPFlowFunc == nil {
		utils.Debug(func() {
			log.Warn("SaveResponse failed because yakit.RegisterSaveHTTPFlowHandler is not finished")
		})
		return
	}
	rawPacket := r.RawPacket
	r.HiddenIndex = uuid.NewString()

	if r.TooLarge {
		rawPacket = ReplaceHTTPPacketBodyFast(rawPacket, []byte(`[[response too large(`+utils.ByteSize(uint64(r.TooLargeLimit))+`), truncated]] find more in web fuzzer history!`))
	}
	saveHTTPFlowFunc(r.Https, r.RawRequest, rawPacket, r.Url, r.RemoteAddr, r.Source, r.RuntimeId, r.FromPlugin, r.HiddenIndex, r.Payloads)
}
