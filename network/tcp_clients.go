package network

import (
	"net"
	"net/netip"
	"sync"

	"sirherobrine23.org/playit-cloud/go-playit/tunnel"
	"sirherobrine23.org/playit-cloud/go-playit/proto"
)

type ActiveClients struct {
	locked sync.Mutex
	// active: Arc<RwLock<HashMap<(SocketAddr, SocketAddr), NewClient>>>,
	active map[[2]netip.AddrPort]proto.NewClient
}

func NewActiveClients() ActiveClients {
	return ActiveClients{
		locked: sync.Mutex{},
		active: make(map[[2]netip.AddrPort]proto.NewClient),
	}
}

func (clients *ActiveClients) Len() int {
	return len(clients.active)
}

func (clients *ActiveClients) GetClients() []proto.NewClient {
	clientsArr := []proto.NewClient{}
	for _, cl := range clients.active {
		clientsArr = append(clientsArr, cl)
	}
	return clientsArr
}

func (clients *ActiveClients) AddNew(client proto.NewClient) *Dropper {
	clients.locked.Lock()
	defer clients.locked.Unlock()
	for actClient := range clients.active {
		if client.PeerAddr.Compare(actClient[0]) == 0 && client.ConnectAddr.Compare(actClient[1]) == 0 {
			return nil
		}
	}
	key := [2]netip.AddrPort{client.PeerAddr, client.ConnectAddr}
	clients.active[key] = client
	return &Dropper{key, *clients}
}

type Dropper struct {
	key   [2]netip.AddrPort
	inner ActiveClients
}

func (dr *Dropper) Drop() {
	PeerAddr, ConnectAddr := dr.key[0], dr.key[1]
	dr.inner.locked.Lock()
	defer dr.inner.locked.Unlock()
	for client := range dr.inner.active {
		if client[0].Compare(PeerAddr) == 0 && client[1].Compare(ConnectAddr) == 0 {
			delete(dr.inner.active, client)
			break
		}
	}
}

func NewTcpClients() TcpClients {
	return TcpClients{NewActiveClients(), true}
}

type TcpClients struct {
	active        ActiveClients
	UseSpecialLAN bool
}

func (tcp *TcpClients) ActiveClients() ActiveClients {
	return tcp.active
}

func (tcp *TcpClients) Connect(newClient proto.NewClient) (*TcpClient, error) {
	claimInstruction := newClient.ClaimInstructions
	droppe := tcp.active.AddNew(newClient)
	if droppe == nil {
		return nil, nil
	}

	stream, err := (&tunnel.TcpTunnel{claimInstruction}).Connect()
	if err != nil {
		return nil, err
	}

	return &TcpClient{*stream, *droppe}, nil
}

type TcpClient struct {
	Stream  net.TCPConn
	Dropper Dropper
}
