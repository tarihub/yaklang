package cybertunnel

import (
	"context"
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/google/uuid"
	"github.com/yaklang/yaklang/common/cybertunnel/tpb"
	"github.com/yaklang/yaklang/common/log"
	"github.com/yaklang/yaklang/common/utils"
	"github.com/yaklang/yaklang/common/utils/lowhttp/poc"
	"google.golang.org/grpc"
	"net"
	"strings"
	"testing"
	"time"
)

func CreateCyberTunnelLocalClient(domain string) (tpb.TunnelClient, tpb.TunnelServer) {
	s, err := NewTunnelServer(domain, "127.0.0.1")
	if err != nil {
		panic(err)
	}

	trans := grpc.NewServer()
	tpb.RegisterTunnelServer(trans, s)

	port := utils.GetRandomAvailableTCPPort()
	lis, err := net.Listen("tcp", utils.HostPort("127.0.0.1", port))
	go func() {
		err := trans.Serve(lis)
		if err != nil {
			panic(err)
		}
	}()
	time.Sleep(time.Second)

	conn, err := grpc.Dial(
		utils.HostPort("127.0.0.1", port),
		grpc.WithInsecure(),
		grpc.WithNoProxy(),
	)
	if err != nil {
		panic(err)
	}
	client := tpb.NewTunnelClient(conn)
	return client, s
}

func TestHTTPTrigger_Register(t *testing.T) {
	trigger, err := NewHTTPTrigger("127.0.0.1", "test")
	if err != nil {
		t.Fatal(err)
	}

	httpPort := utils.GetRandomAvailableTCPPort()
	httpsPort := utils.GetRandomAvailableTCPPort()
	trigger.SetHTTPPort(httpPort)
	trigger.SetHTTPSPort(httpsPort)

	client, server := CreateCyberTunnelLocalClient("test")
	_ = server
	go func() {
		trigger.Serve()
	}()
	err = utils.WaitConnect("127.0.0.1:"+fmt.Sprint(httpPort), 4)
	if err != nil {
		t.Fatal(err)
	}
	defaultHTTPTrigger = trigger
	uid := uuid.New().String()
	rsp, err := client.RequireHTTPRequestTrigger(
		context.Background(),
		&tpb.RequireHTTPRequestTriggerParams{
			ExpectedHTTPResponse: []byte("HTTP/1.1 302 Found\r\n" + "Location: " + uid + "\r\nContent-Length: 0\r\n\r\n"),
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(rsp.Urls) <= 0 {
		t.Fatal("no urls")
	}
	u := rsp.Urls[0]
	if strings.HasPrefix(u, "https://") {
		u = "http://" + u[8:]
	}
	httpResponse, _, err := poc.DoGET(
		u, poc.WithHost("127.0.0.1"), poc.WithPort(httpPort),
		poc.WithNoRedirect(true),
	)
	if err != nil {
		log.Warnf("GET %v failed: %v", u, err)
		t.Fatal(err)
	}
	_ = httpResponse
	token := rsp.GetToken()
	spew.Dump(token)
	notifResponse, err := client.QueryExistedHTTPRequestTrigger(context.Background(), &tpb.QueryExistedHTTPRequestTriggerRequest{Token: token})
	if err != nil {
		t.Fatal(err)
	}
	ns := notifResponse.GetNotifications()
	if len(ns) <= 0 {
		t.Fatal("no notifications")
	}
	packet := string(httpResponse.RawPacket)
	if !strings.Contains(packet, "Location: "+uid+"\r\n") {
		t.Fatal("no uid included")
	}

	notifResponse, err = client.QueryExistedHTTPRequestTrigger(context.Background(), &tpb.QueryExistedHTTPRequestTriggerRequest{Token: uuid.New().String()})
	if err != nil {
		t.Fatal(err)
	}
	if len(notifResponse.Notifications) > 0 {
		t.Fatal("should be empty")
	}
}
