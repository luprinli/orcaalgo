package notify

type PushNotifier struct {
	hub     PushHub
	enabled bool
}

func NewPushNotifier(hub PushHub) *PushNotifier {
	return &PushNotifier{
		hub:     hub,
		enabled: hub != nil,
	}
}

func (p *PushNotifier) Name() string {
	return "push"
}

func (p *PushNotifier) IsEnabled() bool {
	return p.enabled
}

func (p *PushNotifier) Send(event Event) error {
	if !p.enabled {
		return nil
	}
	p.hub.Broadcast("notification", event)
	p.hub.Broadcast("notification_"+string(event.Type), event)
	return nil
}
