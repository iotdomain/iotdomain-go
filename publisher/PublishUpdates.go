// Package publisher with publication of updates of registered entities
package publisher

import (
	"time"

	"github.com/iotdomain/iotdomain-go/inputs"
	"github.com/iotdomain/iotdomain-go/lib"
	"github.com/iotdomain/iotdomain-go/messaging"
	"github.com/iotdomain/iotdomain-go/nodes"
	"github.com/iotdomain/iotdomain-go/outputs"
	"github.com/iotdomain/iotdomain-go/types"
	"github.com/sirupsen/logrus"
)

// PublishUpdates publishes changes to registered nodes, inputs, outputs, values and this publisher identity
func (publisher *Publisher) PublishUpdates() {

	updatedNodes := publisher.registeredNodes.GetUpdatedNodes(true)
	nodes.PublishRegisteredNodes(updatedNodes, publisher.messageSigner)
	if len(updatedNodes) > 0 && publisher.config.ConfigFolder != "" {
		publisher.SaveRegisteredNodes()
	}

	updatedInputs := publisher.registeredInputs.GetUpdatedInputs(true)
	inputs.PublishRegisteredInputs(updatedInputs, publisher.messageSigner)

	updatedOutputs := publisher.registeredOutputs.GetUpdatedOutputs(true)
	outputs.PublishRegisteredOutputs(updatedOutputs, publisher.messageSigner)

	updatedOutputIDs := publisher.registeredOutputValues.GetUpdatedOutputValues(true)
	publisher.PublishUpdatedOutputValues(updatedOutputIDs, publisher.messageSigner)
}

// PublishUpdatedOutputValues publishes updated outputs discovery and values of registered outputs
// This uses the node config to determine which output publications to use: eg raw, latest, history
func (publisher *Publisher) PublishUpdatedOutputValues(
	updatedOutputIDs []string,
	messageSigner *messaging.MessageSigner) {
	regOutputValues := publisher.registeredOutputValues

	for _, outputID := range updatedOutputIDs {
		var node *types.NodeDiscoveryMessage
		latestValue := regOutputValues.GetOutputValueByID(outputID)
		output := publisher.registeredOutputs.GetOutputByID(outputID)

		if output == nil {
			logrus.Warningf("PublishOutputValues: output with ID %s. This is unexpected", outputID)
		} else {
			node = publisher.registeredNodes.GetNodeByHWID(output.NodeHWID)
		}
		if node == nil {
			logrus.Warningf("PublishOutputValues: no node for output %s. This is unexpected", outputID)
		} else if latestValue == nil {
			logrus.Warningf("PublishOutputValues: no latest value for %s. This is unexpected", outputID)
		} else {
			pubRaw, _ := publisher.registeredNodes.GetNodeConfigBool(node.Address, types.NodeAttrPublishRaw, true)
			if pubRaw {
				outputs.PublishOutputRaw(output, latestValue.Value, messageSigner)
			}
			pubLatest, _ := publisher.registeredNodes.GetNodeConfigBool(node.Address, types.NodeAttrPublishLatest, true)
			if pubLatest {
				outputs.PublishOutputLatest(output, latestValue, messageSigner)
			}
			pubHistory, _ := publisher.registeredNodes.GetNodeConfigBool(node.Address, types.NodeAttrPublishHistory, true)
			if pubHistory {
				history := regOutputValues.GetHistory(outputID)
				outputs.PublishOutputHistory(output, history, messageSigner)
			}
			pubEvent, _ := publisher.registeredNodes.GetNodeConfigBool(node.Address, types.NodeAttrPublishEvent, false)
			if pubEvent {
				PublishOutputEvent(node, publisher.registeredOutputs, publisher.registeredOutputValues, messageSigner)
			}
		}
	}
}

// PublishOutputEvent publishes all node output values in the $event command
// zone/publisher/nodealias/$event
// TODO: decide when to invoke this
func PublishOutputEvent(
	node *types.NodeDiscoveryMessage,
	registeredOutputs *outputs.RegisteredOutputs,
	outputValues *outputs.RegisteredOutputValues,
	messageSigner *messaging.MessageSigner,
) error {
	// output values are published using their alias address, if any
	aliasAddress := outputs.ReplaceMessageType(node.Address, types.MessageTypeEvent)
	logrus.Infof("Publisher.publishEvent: %s", aliasAddress)

	nodeOutputs := registeredOutputs.GetOutputsByNodeHWID(node.HWID)
	event := make(map[string]string)
	timeStampStr := time.Now().Format("2006-01-02T15:04:05.000-0700")
	if len(nodeOutputs) == 0 {
		return lib.MakeErrorf("PublishOutputEvent: Node %s doesn't have any outputs", node.Address)
	}
	for _, output := range nodeOutputs {
		var value = ""
		latest := outputValues.GetOutputValueByID(output.OutputID)
		attrID := string(output.OutputType) + "/" + output.Instance
		if latest != nil {
			value = latest.Value
		}
		event[attrID] = value
	}
	eventMessage := &types.OutputEventMessage{
		Address:   aliasAddress,
		Event:     event,
		Timestamp: timeStampStr,
	}
	err := messageSigner.PublishObject(aliasAddress, true, eventMessage, nil)
	return err
}
