package core

import (
	"barrel-api/model"
	"barrel-api/repository"
	"encoding/json"
	"fmt"
	"log"

	pahomqtt "github.com/eclipse/paho.mqtt.golang"
)

// RoutineTriggerListener subscribes to device status topics and fires
// routines whose trigger matches the incoming device state.
type RoutineTriggerListener struct {
	routineRepo repository.RoutineRepositoryInterface
	executor    RoutineRunnerIface
	brokerURL   string
	adminUser   string
	adminPass   string
	client      pahomqtt.Client
}

func NewRoutineTriggerListener(
	routineRepo repository.RoutineRepositoryInterface,
	executor RoutineRunnerIface,
	brokerURL, adminUser, adminPass string,
) *RoutineTriggerListener {
	return &RoutineTriggerListener{
		routineRepo: routineRepo,
		executor:    executor,
		brokerURL:   brokerURL,
		adminUser:   adminUser,
		adminPass:   adminPass,
	}
}

// Start connects to the MQTT broker and subscribes to all device status topics.
func (l *RoutineTriggerListener) Start() error {
	opts := pahomqtt.NewClientOptions().
		AddBroker(l.brokerURL).
		SetClientID("barrel-routine-trigger-listener").
		SetUsername(l.adminUser).
		SetPassword(l.adminPass).
		SetAutoReconnect(true).
		SetOnConnectHandler(func(c pahomqtt.Client) {
			log.Println("[routine-listener] connected to MQTT broker")
			l.subscribeAll(c)
		}).
		SetConnectionLostHandler(func(c pahomqtt.Client, err error) {
			log.Printf("[routine-listener] connection lost: %v", err)
		})

	l.client = pahomqtt.NewClient(opts)
	if tok := l.client.Connect(); tok.Wait() && tok.Error() != nil {
		return fmt.Errorf("routine trigger listener: mqtt connect: %w", tok.Error())
	}
	return nil
}

// Stop disconnects gracefully.
func (l *RoutineTriggerListener) Stop() {
	if l.client != nil && l.client.IsConnected() {
		l.client.Disconnect(500)
	}
}

func (l *RoutineTriggerListener) subscribeAll(c pahomqtt.Client) {
	// Subscribe to all device status topics under all users
	tok := c.Subscribe("users/+/+/status", 1, l.handleStatusMessage)
	if tok.Wait() && tok.Error() != nil {
		log.Printf("[routine-listener] subscribe error: %v", tok.Error())
	}
}

// handleStatusMessage is called for every device status update.
// It loads all enabled device-trigger routines and evaluates their conditions.
func (l *RoutineTriggerListener) handleStatusMessage(_ pahomqtt.Client, msg pahomqtt.Message) {
	var state map[string]string
	if err := json.Unmarshal(msg.Payload(), &state); err != nil {
		// Status may be a plain string (e.g. "on"/"off") — wrap it
		state = map[string]string{"power": string(msg.Payload())}
	}

	routines, err := l.routineRepo.GetEnabledRoutinesByTriggerType("device")
	if err != nil {
		log.Printf("[routine-listener] failed to load device routines: %v", err)
		return
	}

	for _, routine := range routines {
		if l.matches(routine, state) {
			go func(r model.Routine) {
				if err := l.executor.ExecuteRoutine(&r); err != nil {
					log.Printf("[routine-listener] execution error (routine=%d): %v", r.ID, err)
				}
			}(routine)
		}
	}
}

// matches checks whether the received device state satisfies the routine trigger condition.
func (l *RoutineTriggerListener) matches(routine model.Routine, receivedState map[string]string) bool {
	expected := routine.Trigger.ExpectedState
	if len(expected) == 0 {
		return false
	}
	for k, v := range expected {
		if receivedState[k] != v {
			return false
		}
	}
	return true
}
