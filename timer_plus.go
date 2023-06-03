package gomicrobee

import "time"

// Timer Plus.
// Like time.Timer with improved stop behaviour.
//
// To check if the timer is stopped, read from the stopCh.
// Unlike time.Timer.Stop the consumer will not get stuck
// checking stopped, when its stopped.

type timer_plus struct {
	timer  *time.Timer
	stopCh chan bool
}

func newTimerPlus(d time.Duration) timer_plus {
	t := time.NewTimer(d)
	return timer_plus{t, make(chan bool, 1)}
}

func (t *timer_plus) stop() {
	t.stopCh <- true
	t.timer.Stop()
}
