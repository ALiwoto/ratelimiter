// ratelimiter Project
// Copyright (C) 2021~2022 ALiwoto and other Contributors
// This file is subject to the terms and conditions defined in
// file 'LICENSE', which is part of the source code.

package ratelimiter

import (
	"sync"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers/filters"
)

//---------------------------------------------------------

// Start will start the limiter.
// When the limiter is started (enabled), it will check for
// check for incoming messages; if they are considered as flood,
// the limiter won't let the handler functions to be called.
func (l *Limiter) Start() {
	if l.isEnabled {
		return
	}

	if l.mutex == nil {
		l.mutex = new(sync.RWMutex)
	}

	if l.userMap == nil {
		l.userMap = make(map[int64]*UserStatus)
	}

	l.isEnabled = true
	l.isStopped = false

	go l.checker()
}

// Stop method will make this limiter stop checking the incoming
// messages and will set its variables to nil.
// the main resources used by this limiter will be freed,
// such as map and mutex.
// but the configuration variables such as message time out will
// remain the same and won't be set to 0.
func (l *Limiter) Stop() {
	if l.isStopped {
		return
	}

	l.isEnabled = false
	l.isStopped = true

	// make sure that mutex is not nil.
	if l.mutex != nil {
		// let another goroutines let go of the mutex;
		// if you set the userMap value to nil out of nowhere
		// it MAY cause some troubles.
		l.mutex.Lock()
		l.userMap = nil
		l.mutex.Unlock()

		l.mutex = nil
	}
}

// IsStopped returns true if this limiter is already stopped
// and doesn't check for incoming messages.
func (l *Limiter) IsStopped() bool {
	return l.isStopped
}

// IsEnabled returns true if and only if this limiter is enabled
// and is checking the incoming messages for floodwait.
// for enabling the limiter, you need to use `Start` method.
func (l *Limiter) IsEnabled() bool {
	return l.isEnabled
}

// SetTriggerFuncs will set the trigger functions of this limiter.
// The trigger functions will be triggered when the limiter
// limits a user. The information passed by it will be the
// information related to the last message of the user.
func (l *Limiter) SetTriggerFuncs(t ...handlers.Response) {
	l.triggers = t
}

// SetTriggerFunc will set the trigger function of this limiter.
// The trigger function will be triggered when the limiter
// limits a user. The information passed by it will be the
// information related to the last message of the user.
// If you want to set more than one trigger function, use
// `SetTriggerFuncs` method.
func (l *Limiter) SetTriggerFunc(t handlers.Response) {
	l.SetTriggerFuncs(t)
}

// AppendTriggerFuncs will append trigger functions to the trigger
// functions list of this limiter.
func (l *Limiter) AppendTriggerFuncs(t ...handlers.Response) {
	l.triggers = append(l.triggers, t...)
}

// AppendTriggerFunc will append a trigger function to the trigger
// functions list of this limiter.
func (l *Limiter) AppendTriggerFunc(t handlers.Response) {
	l.triggers = append(l.triggers, t)
}

// AddException will add an exception filter to this limiter.
func (l *Limiter) AddException(ex filters.Message) {
	l.exceptions = append(l.exceptions, ex)
}

// ClearAllExceptions will clear all exception of this limiter.
// this way, you will be sure that all of incoming updates will be
// checked for floodwait by this limiter.
func (l *Limiter) ClearAllExceptions() {
	l.exceptions = nil
}

// GetExceptions returns the filters array used by this limiter as
// its exceptions list.
func (l *Limiter) GetExceptions() []filters.Message {
	return l.exceptions
}

// IsTextOnly will return true if and only if this limiter is
// checking for text-only messages.
func (l *Limiter) IsTextOnly() bool {
	return l.TextOnly
}

// SetTextOnly will set the limiter to check for text-only messages.
// pass true to this method to make the limiter check for text-only
// messages.
func (l *Limiter) SetTextOnly(t bool) {
	l.TextOnly = t
}

// IsAllowingChannels will return true if and only if this limiter
// is checking for messages from channels.
func (l *Limiter) IsAllowingChannels() bool {
	if l.msgHandler == nil {
		return false
	}
	return l.msgHandler.AllowChannel
}

// IsAllowingEdits will return true if and only if this limiter
// is checking for "edited message" update from telegram.
func (l *Limiter) IsAllowingEdits() bool {
	if l.msgHandler == nil {
		return false
	}
	return l.msgHandler.AllowEdited
}

// AddExceptionID will add a group/user/channel ID to the exception
// list of the limiter.
func (l *Limiter) AddExceptionID(id ...int64) {
	l.exceptionIDs = append(l.exceptionIDs, id...)
}

// AddCondition will add a condition to be checked by this limiter,
// if this condition doesn't return true, the limiter won't check
// the message for anti-flood-wait.
func (l *Limiter) AddCondition(condition filters.Message) {
	l.conditions = append(l.conditions, condition)
}

// ClearAllConditions clears all condition list.
func (l *Limiter) ClearAllConditions() {
	l.conditions = nil
}

// AddConditions will accept an array of the conditions and will
// add them to the condition list of this limiter.
// you can also pass only one value to this method.
func (l *Limiter) AddConditions(conditions ...filters.Message) {
	l.conditions = append(l.conditions, conditions...)
}

// SetAsConditions will accept an array of conditions and will set
// the conditions of the limiter to them.
func (l *Limiter) SetAsConditions(conditions []filters.Message) {
	l.conditions = conditions
}

// ClearAllExceptions will clear all exception IDs of this limiter.
// this way, you will be sure that all of incoming updates will be
// checked for floodwait by this limiter.
func (l *Limiter) ClearAllExceptionIDs() {
	l.exceptionIDs = nil
}

// IsInExceptionList will check and see if an ID is in the
// exception list of the listener or not.
func (l *Limiter) IsInExceptionList(id int64) bool {
	if len(l.exceptionIDs) == 0 {
		return false
	}

	for _, ex := range l.exceptionIDs {
		if ex == id {
			return true
		}
	}

	return false
}

// SetAsExceptionList will set its argument at the exception
// list of this limiter. Please notice that this method won't
// append the list to the already existing exception list; but
// it will set it to this, so the already existing exception IDs
// assigned to this limiter will be lost.
func (l *Limiter) SetAsExceptionList(list []int64) {
	l.exceptionIDs = list
}

// GetStatus will get the status of a chat.
// if `l.ConsiderUser` parameter is set to `true`,
// the id should be the id of the user; otherwise you should
// use the id of the chat to get the status.
func (l *Limiter) GetStatus(id int64) *UserStatus {
	var status *UserStatus
	l.mutex.RLock()
	status = l.userMap[id]
	l.mutex.RUnlock()

	return status
}

// SetFloodWaitTime will set the flood wait duration for each
// chat to send `maxCount` message per this amount of time.
// if they send more than this amount of messages during this time,
// they will be limited by this limiter and so their messages
// won't be handled in the current group.
// (Notice: if `ConsiderUser` is set to `true`, this duration will
// be applied to unique users in the chat; not the total chat.)
func (l *Limiter) SetFloodWaitTime(d time.Duration) {
	l.timeout = d
}

// SetPunishmentDuration will set the punishment duration of
// the chat (or a user) after being limited by this limiter.
// Users needs to spend this amount of time + `l.timeout` to become
// free and so the handlers will work again for it.
// NOTICE: if `IsStrict` is set to `true`, as long as user sends
// message to the bot, the amount of passed-punishment time will
// become 0; so the user needs to stop sending messages to the bot
// until the punishment time is passed, otherwise the user will be
// limited forever.
func (l *Limiter) SetPunishmentDuration(d time.Duration) {
	l.punishment = d
}

// SetMaxMessageCount sets the possible messages count in the
// anti-flood-wait amount of time (which is `l.timeout`).
// in that period of time, chat (or user) needs to send less than
// this much message, otherwise they will be limited by this limiter
// and so as a result of that their messages will be ignored by the bot.
func (l *Limiter) SetMaxMessageCount(count int) {
	l.maxCount = count
}

// SetMaxCacheDuration will set the max duration for caching algorithm.
// WARNING: this value should always be greater than the
// `timeout` + `punishment` values of the limiter;
// otherwise this method will set the max cache duration to
// `timeout` + `punishment` + 1.
func (l *Limiter) SetMaxCacheDuration(d time.Duration) {
	if d > l.punishment+l.timeout {
		l.maxTimeout = d
	} else {
		l.maxTimeout = l.punishment + l.timeout + time.Minute
	}
}

// SetDefaultInterval will set a default value to the checker's interval.
// It's recommended that users use `SetMaxCacheDuration` method instead of this one.
// If you haven't set any other parameters for the limiter, this will set the interval
// to 60 seconds at least.
func (l *Limiter) SetDefaultInterval() {
	l.maxTimeout = l.punishment + l.timeout + time.Minute
}

func (l *Limiter) AddCustomIgnore(id int64, d time.Duration, ignoreExceptions bool) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	status := l.userMap[id]
	if status == nil {
		status = new(UserStatus)
		status.custom = &customIgnore{
			startTime:       time.Now(),
			duration:        d,
			ignoreException: ignoreExceptions,
		}
		l.userMap[id] = status
		if ignoreExceptions {
			l.addIgnoredExceptions(id)
		}

		return
	}

	status.custom = &customIgnore{
		startTime:       time.Now(),
		duration:        d,
		ignoreException: ignoreExceptions,
	}
	if ignoreExceptions {
		l.addIgnoredExceptions(id)
	}
}

func (l *Limiter) RemoveCustomIgnore(id int64) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	status := l.userMap[id]
	if status == nil || status.custom == nil {
		return
	}

	if status.custom.ignoreException {
		l.removeFromIgnoredExceptions(id)
	}
	status.custom = nil
}

// hasTextCondition will check if the message meets the message condition
// or not.
// basically if l.TextOnly is set to true, this method will check if
// the message is a normal text message or not.
func (l *Limiter) hasTextCondition(msg *gotgbot.Message) bool {
	if l.TextOnly {
		return len(msg.Text) > 0
	}

	return true
}

// runTriggers will run the triggers of the limiter.
// this method should be called in a separate goroutine.
func (l *Limiter) runTriggers(b *gotgbot.Bot, ctx *ext.Context) {
	for _, trigger := range l.triggers {
		if trigger != nil {
			trigger(b, ctx)
		}
	}
}

// isException will check and see if msg can be ignored because
// it's id is in the exception list or not. This method's usage
// is internal-only.
func (l *Limiter) isException(msg *gotgbot.Message) bool {
	if len(l.exceptionIDs) == 0 || msg == nil {
		return false
	}

	for _, ex := range l.exceptionIDs {
		if msg.From != nil {
			if ex == msg.From.Id || ex == msg.Chat.Id {
				return true
			}
		} else {
			if ex == msg.Chat.Id {
				return true
			}
		}

	}

	return false
}

func (l *Limiter) isExceptionCtx(ctx *ext.Context) bool {
	if ctx.CallbackQuery != nil {
		return l.isExceptionQuery(ctx.CallbackQuery)
	}
	return l.isException(ctx.Message)
}

// isException will check and see if msg can be ignored because
// it's id is in the exception list or not. This method's usage
// is internal-only.
func (l *Limiter) isExceptionQuery(cq *gotgbot.CallbackQuery) bool {
	if len(l.exceptionIDs) == 0 || cq == nil {
		return false
	}

	for _, ex := range l.exceptionIDs {
		if ex == cq.From.Id || (cq.Message != nil && ex == cq.Message.GetChat().Id) {
			return true
		}
	}

	return false
}

// isIgnoredException will check and see if msg cannot be ignored because
// it's id is in the exception list or not. This method's usage
// is internal-only.
func (l *Limiter) isIgnoredException(msg *gotgbot.Message) bool {
	if len(l.ignoredExceptions) == 0 {
		return false
	}

	for _, ex := range l.ignoredExceptions {
		if msg.From != nil {
			if ex == msg.From.Id || ex == msg.Chat.Id {
				return true
			}
		} else {
			if ex == msg.Chat.Id {
				return true
			}
		}
	}

	return false
}

// isIgnoredException will check and see if msg cannot be ignored because
// it's id is in the exception list or not. This method's usage
// is internal-only.
func (l *Limiter) isIgnoredExceptionQuery(cq *gotgbot.CallbackQuery) bool {
	if len(l.ignoredExceptions) == 0 {
		return false
	}

	for _, ex := range l.ignoredExceptions {
		if ex == cq.From.Id || (cq.Message != nil && ex == cq.Message.GetChat().Id) {
			return true
		}
	}

	return false
}

func (l *Limiter) addIgnoredExceptions(id int64) {
	if len(l.ignoredExceptions) == 0 {
		l.ignoredExceptions = append(l.ignoredExceptions, id)
		return
	}
	for _, ex := range l.ignoredExceptions {
		if ex == id {
			return
		}
	}
	l.ignoredExceptions = append(l.ignoredExceptions, id)
}

func (l *Limiter) removeFromIgnoredExceptions(id int64) {
	if len(l.ignoredExceptions) == 0 {
		return
	}
	for i, ex := range l.ignoredExceptions {
		if ex == id {
			l.ignoredExceptions = append(l.ignoredExceptions[:i], l.ignoredExceptions[i+1:]...)
			return
		}
	}
}

// checker should be run in a new goroutine as it blocks its goroutine
// with a for-loop. This method's duty is to clear the old user's status
// from the cache using `l.maxTimeout` parameter.
func (l *Limiter) checker() {
	for l.isEnabled && !l.isStopped {
		if l.maxTimeout < time.Second {
			// if we don't do this, we will end up running an unlimited
			// loop with highest possible speed (which will cause high
			// cpu usage).
			l.SetDefaultInterval()
		}
		time.Sleep(l.maxTimeout)

		// added this checker just in-case so we can
		// prevent the panics in the future.
		if l.userMap == nil || l.mutex == nil {
			// return from the cleaner function and let the
			// goroutine die.
			return
		}

		if len(l.userMap) == 0 {
			continue
		}

		l.mutex.Lock()
		for key, value := range l.userMap {
			if value == nil || value.canBeDeleted(l) {
				delete(l.userMap, key)
			}
		}
		l.mutex.Unlock()
	}
}

//---------------------------------------------------------

// IsLimited will check and see if the chat (or user) is
// limited by this limiter or not.
func (s *UserStatus) IsLimited() bool {
	return s.limited
}

func (s *UserStatus) IsCustomLimited() bool {
	if s.custom == nil {
		return false
	}

	if time.Since(s.custom.startTime) > s.custom.duration && s.custom.duration != 0 {
		s.custom = nil
		return false
	}

	return true
}

func (s *UserStatus) canBeDeleted(l *Limiter) bool {
	return s.Last.IsZero() ||
		(time.Since(s.Last) > l.timeout && !s.limited && !s.IsCustomLimited())
}

//---------------------------------------------------------
