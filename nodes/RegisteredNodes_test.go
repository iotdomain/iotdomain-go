package nodes_test

import (
	"crypto/ecdsa"
	"fmt"
	"testing"

	"github.com/iotdomain/iotdomain-go/messaging"
	"github.com/iotdomain/iotdomain-go/nodes"
	"github.com/iotdomain/iotdomain-go/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const domain = "test"

const node1ID = "node1"
const publisher1ID = "publisher1"
const publisher2ID = "publisher2"

var node1Base = fmt.Sprintf("%s/%s/%s", domain, publisher1ID, node1ID)
var node1Addr = node1Base + "/$node"

var node1ConfigureAddr = node1Base + "/$configure"
var node1InputAddr = node1Base + "/switch/0/$input"
var node1InputSetAddr = node1Base + "/switch/0/$set"

var node1Output1Addr = node1Base + "/switch/0/$output"
var node1Output1Type = "switch"
var node1Output1Instance = "0"

var node1valueAddr = node1Base + "/switch/0/$value"
var node1latestAddr = node1Base + "/switch/0/$latest"
var node1historyAddr = node1Base + "/switch/0/$history"

// const node2 = new node.Node{}

// TestNewNode instance
func TestNewNode(t *testing.T) {
	collection := nodes.NewRegisteredNodes(domain, publisher1ID)
	node := collection.CreateNode(node1ID, types.NodeTypeUnknown)

	if !assert.NotNil(t, node, "Failed creating node") {
		return
	}
	node2 := collection.GetNodeByAddress(node1Addr)
	require.NotNil(t, node2, "Failed getting created node")
	node2b := collection.GetNodeByID(node1ID)
	require.NotNil(t, node2b)
	node3 := collection.GetNodeByAddress("not/valid")
	require.Nil(t, node3, "Get node invalid address")

	node1b := collection.CreateNode(node1ID, types.NodeTypeUnknown)
	assert.Equal(t, node2, node1b, "Creating the same node twice should return the existing node")

	all := collection.GetAllNodes()
	assert.Equal(t, 1, len(all))
	updated := collection.GetUpdatedNodes(true)
	assert.Equal(t, 1, len(updated))

	// update list of nodes
	nodeList := make([]*types.NodeDiscoveryMessage, 0)
	newNode := nodes.NewNode(domain, publisher2ID, "device2", "unknown")
	newNode.Attr = nil
	newNode.Status = nil
	newNode.Config = nil
	nodeList = append(nodeList, newNode)
	collection.UpdateNodes(nodeList)
	all = collection.GetAllNodes()
	assert.Equal(t, 2, len(all))

}

// Test updating of node atributes and status
func TestAttrStatus(t *testing.T) {
	const node1ID = "node1"
	collection := nodes.NewRegisteredNodes(domain, publisher1ID)
	collection.CreateNode(node1ID, types.NodeTypeUnknown)

	newAttr := map[types.NodeAttr]string{types.NodeAttrManufacturer: "Bob"}
	collection.UpdateNodeAttr(node1ID, newAttr)

	newStatus := map[types.NodeStatus]string{"LastUpdated": "now"}
	collection.UpdateNodeStatus(node1ID, newStatus)

	node1 := collection.GetNodeByAddress(node1Addr)
	val, hasAttr := node1.Attr[types.NodeAttrManufacturer]
	require.True(t, hasAttr, "Can't find attribute Manufacturer")
	assert.Equal(t, "Bob", val, "Attribute change wasn't applied")
	// get attr
	attr := collection.GetNodeAttr(node1ID, types.NodeAttrManufacturer)
	assert.NotEmpty(t, attr, "Attr not found")
	attr = collection.GetNodeAttr("unknownnode", types.NodeAttrManufacturer)
	assert.Empty(t, attr, "Attr found")

}

// TestConfigure tests if the node configuration is handled
func TestConfigure(t *testing.T) {
	collection := nodes.NewRegisteredNodes(domain, publisher1ID)
	node := collection.CreateNode(node1ID, types.NodeTypeUnknown)

	config := collection.NewNodeConfig(node1ID, types.NodeAttrName, types.DataTypeString, "Friendly Name", "bob")
	collection.UpdateNodeConfig(node1ID, types.NodeAttrName, config)
	// string
	c, err := collection.GetNodeConfigString(node1ID, types.NodeAttrName, "def")
	assert.NoError(t, err)
	assert.Equal(t, "bob", c)
	c, err = collection.GetNodeConfigString("invlidNode", types.NodeAttrName, "def")
	assert.Error(t, err)
	assert.Equal(t, "def", c) // use default value
	// bool
	_, err = collection.GetNodeConfigBool(node1ID, types.NodeAttrName, false)
	assert.Error(t, err) // not a bool
	_, err = collection.GetNodeConfigBool("notanode", types.NodeAttrName, false)
	assert.Error(t, err) // no node
	config = collection.NewNodeConfig(node1ID, types.NodeAttrDisabled, types.DataTypeString, "Node is disabled", "false")
	collection.UpdateNodeConfig(node1ID, types.NodeAttrDisabled, config)
	_, err = collection.GetNodeConfigBool(node1ID, types.NodeAttrDisabled, false)
	assert.NoError(t, err) // no node
	// float
	_, err = collection.GetNodeConfigFloat(node1ID, types.NodeAttrMin, 1.1)
	assert.Error(t, err) // not a number
	config = collection.NewNodeConfig(node1ID, types.NodeAttrMin, types.DataTypeNumber, "min", "1.23")
	collection.UpdateNodeConfig(node1ID, types.NodeAttrDisabled, config)
	_, err = collection.GetNodeConfigFloat(node1ID, types.NodeAttrMin, 1.1)
	assert.NoError(t, err)
	// int
	_, err = collection.GetNodeConfigInt(node1ID, types.NodeAttrMin, 0)
	assert.Error(t, err) // float is not an int
	_, err = collection.GetNodeConfigInt("notanode", types.NodeAttrMin, 1)
	assert.Error(t, err) // not a number
	collection.UpdateNodeConfigValues(node1ID, types.NodeAttrMap{types.NodeAttrMin: "2"})
	val2, err := collection.GetNodeConfigInt(node1ID, types.NodeAttrMin, 2)
	assert.Equal(t, 2, val2) // Use default
	assert.NoError(t, err)   // not a number
	// not a config
	changed := collection.UpdateNodeConfigValues("notanode", types.NodeAttrMap{"": "2"})
	assert.False(t, changed)
	changed = collection.UpdateNodeConfigValues(node1ID, types.NodeAttrMap{"": "2"})
	assert.False(t, changed)

	// config values
	newValues := map[types.NodeAttr]string{types.NodeAttrName: "NewName"}
	collection.UpdateNodeConfigValues(node1ID, newValues)

	// node1 must match the newly added node
	node = collection.GetNodeByAddress(node1Addr)
	require.NotNil(t, node, "Node %s not found", node1Addr)

	config2 := node.Config[types.NodeAttrName]
	value2 := node.Attr[types.NodeAttrName]
	if !assert.NotNil(t, config2, "Can't find configuration for name") {
		return
	}
	assert.Equal(t, "NewName", value2, "Configuration wasn't applied")
}

func TestReceiveConfig(t *testing.T) {
	const node1ID = "node1"
	const publisher1ID = "publisher1"
	var privKey = messaging.CreateAsymKeys()

	getPublisherKey := func(addr string) *ecdsa.PublicKey {
		return &privKey.PublicKey
	}
	handler := func(address string, params types.NodeAttrMap) types.NodeAttrMap {
		return params
	}
	collection := nodes.NewRegisteredNodes(domain, publisher1ID)
	node1 := collection.CreateNode(node1ID, types.NodeTypeUnknown)

	msgr := messaging.NewDummyMessenger(nil)
	signer := messaging.NewMessageSigner(true, getPublisherKey, msgr, privKey)
	// receive
	receiver := nodes.NewReceiveNodeConfigure(domain, publisher1ID, nil, signer, collection, privKey)
	receiver.SetConfigureNodeHandler(handler)
	receiver.Start()
	// publish
	nodes.PublishNodeConfigure(node1.Address, types.NodeAttrMap{
		types.NodeAttrName: "bob",
	}, "senderaddress", signer, &privKey.PublicKey)
	receiver.Stop()
	name := collection.GetNodeAttr(node1ID, types.NodeAttrName)
	assert.Equal(t, "bob", name)
}

func TestAlias(t *testing.T) {
	const node1AliasID = "alias1"

	// msgConfig := messaging.MessengerConfig{}
	// var testMessenger = messaging.NewDummyMessenger(&msgConfig)
	collection := nodes.NewRegisteredNodes(domain, publisher1ID)

	collection.CreateNode(node1ID, types.NodeTypeUnknown)
	collection.SetAlias(node1ID, node1AliasID)

	node2 := collection.GetNodeByID(node1AliasID)
	assert.NotNil(t, node2, "Node not found using alias")
	collection.SetAlias(node1AliasID, "") // clear alias
	node2 = collection.GetNodeByID(node1AliasID)
	// assert.Nil(t, node2, "Node found using alias")

	// error cases
	collection.SetAlias("invalid", node1AliasID) // invalid nodeID
	collection.SetAlias(node1AliasID, node1ID)   // ignored as this is an existing device

	var privKey = messaging.CreateAsymKeys()

	getPublisherKey := func(addr string) *ecdsa.PublicKey {
		return &privKey.PublicKey
	}

	msgr := messaging.NewDummyMessenger(nil)
	signer := messaging.NewMessageSigner(true, getPublisherKey, msgr, privKey)
	aliasHandler := func(address string, message *types.NodeAliasMessage) {
		// collection.HandleAlias()
	}
	receiver := nodes.NewReceiveNodeAlias(domain, publisher1ID, nil, signer, privKey)
	receiver.SetAliasHandler(aliasHandler)
	receiver.Start()

	err := nodes.PublishNodeAlias(node1Addr, node1AliasID, "sender", signer, &privKey.PublicKey)
	assert.NoErrorf(t, err, "Publish set alias failed: %s", err)
	node2 = collection.GetNodeByID(node1AliasID)
	assert.NotNil(t, node2, "Node not found using alias")

	receiver.Stop()

}
func TestError(t *testing.T) {
	collection := nodes.NewRegisteredNodes(domain, publisher1ID)
	collection.CreateNode(node1ID, types.NodeTypeUnknown)
	collection.UpdateErrorStatus("notanode", types.NodeRunStateError, "This is an error")
	collection.UpdateErrorStatus(node1ID, types.NodeRunStateError, "This is an error")
}

func TestPublishReceive(t *testing.T) {

	var privKey = messaging.CreateAsymKeys()

	getPublisherKey := func(addr string) *ecdsa.PublicKey {
		return &privKey.PublicKey
	}

	msgr := messaging.NewDummyMessenger(nil)
	signer := messaging.NewMessageSigner(true, getPublisherKey, msgr, privKey)
	collection := nodes.NewRegisteredNodes(domain, publisher1ID)

	node := collection.CreateNode(node1ID, types.NodeTypeUnknown)
	require.NotNil(t, node, "Failed creating node")

	allNodes := collection.GetAllNodes()

	nodes.PublishRegisteredNodes(allNodes, signer)

}
