package core

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

// BotRouter routes DM messages for one bot across multiple switchable projects.
type BotRouter struct {
	BotID          string
	DefaultProject string
	DMOnly         bool
	Catalog        *ProjectCatalog
	Bindings       *BindingStore
	Runtimes       *BotRuntimeManager
	ProjectKey     func(*Message) BindingKey
}

func (r *BotRouter) HandleMessage(p Platform, msg *Message) {
	if r.DMOnly && !msg.IsDM {
		_ = p.Reply(context.Background(), msg.ReplyCtx, "Project switching is only available in direct messages.")
		return
	}

	if strings.HasPrefix(strings.TrimSpace(msg.Content), "/project") {
		r.handleProjectCommand(p, msg)
		return
	}

	key := r.bindingKey(msg)
	rec, ok := r.Bindings.Get(key)
	if (!ok || rec.ActiveProject == "") && r.DefaultProject != "" {
		if _, exists := r.Catalog.Projects[r.DefaultProject]; exists {
			r.Bindings.Set(key, r.DefaultProject)
			rec = BindingRecord{ActiveProject: r.DefaultProject}
			ok = true
		}
	}
	if !ok || rec.ActiveProject == "" {
		_ = p.Reply(context.Background(), msg.ReplyCtx, "No active project. Use /project list and /project switch <name> first.")
		return
	}

	rt, err := r.Runtimes.GetOrCreate(r.BotID, rec.ActiveProject)
	if err != nil {
		_ = p.Reply(context.Background(), msg.ReplyCtx, fmt.Sprintf("Failed to load project %q: %v", rec.ActiveProject, err))
		return
	}
	if rt.Engine == nil {
		_ = p.Reply(context.Background(), msg.ReplyCtx, fmt.Sprintf("Project %q has no engine runtime", rec.ActiveProject))
		return
	}
	rt.Engine.HandleIncomingMessage(p, msg)
}

func (r *BotRouter) bindingKey(msg *Message) BindingKey {
	if r.ProjectKey != nil {
		return r.ProjectKey(msg)
	}
	return BindingKey{
		Platform: msg.Platform,
		UserID:   msg.UserID,
		BotID:    r.BotID,
	}
}

func (r *BotRouter) handleProjectCommand(p Platform, msg *Message) {
	parts := strings.Fields(strings.TrimSpace(msg.Content))
	if len(parts) < 2 {
		_ = p.Reply(context.Background(), msg.ReplyCtx, "Usage: /project [list|current|switch]")
		return
	}

	switch strings.ToLower(parts[1]) {
	case "list":
		r.cmdList(p, msg)
	case "current":
		r.cmdCurrent(p, msg)
	case "switch":
		if len(parts) < 3 {
			_ = p.Reply(context.Background(), msg.ReplyCtx, "Usage: /project switch <name>")
			return
		}
		r.cmdSwitch(p, msg, parts[2])
	default:
		_ = p.Reply(context.Background(), msg.ReplyCtx, "Usage: /project [list|current|switch]")
	}
}

func (r *BotRouter) cmdList(p Platform, msg *Message) {
	if r.Catalog == nil || len(r.Catalog.Projects) == 0 {
		_ = p.Reply(context.Background(), msg.ReplyCtx, "No projects found in workspace.")
		return
	}
	names := make([]string, 0, len(r.Catalog.Projects))
	for name := range r.Catalog.Projects {
		names = append(names, name)
	}
	sort.Strings(names)
	_ = p.Reply(context.Background(), msg.ReplyCtx, "Projects:\n- "+strings.Join(names, "\n- "))
}

func (r *BotRouter) cmdCurrent(p Platform, msg *Message) {
	rec, ok := r.Bindings.Get(r.bindingKey(msg))
	if !ok || rec.ActiveProject == "" {
		_ = p.Reply(context.Background(), msg.ReplyCtx, "No active project.")
		return
	}
	_ = p.Reply(context.Background(), msg.ReplyCtx, fmt.Sprintf("Current project: %s", rec.ActiveProject))
}

func (r *BotRouter) cmdSwitch(p Platform, msg *Message, project string) {
	if r.Catalog == nil {
		_ = p.Reply(context.Background(), msg.ReplyCtx, "No workspace catalog loaded.")
		return
	}
	if _, ok := r.Catalog.Projects[project]; !ok {
		_ = p.Reply(context.Background(), msg.ReplyCtx, fmt.Sprintf("Project %q not found.", project))
		return
	}
	r.Bindings.Set(r.bindingKey(msg), project)
	_, err := r.Runtimes.GetOrCreate(r.BotID, project)
	if err != nil {
		_ = p.Reply(context.Background(), msg.ReplyCtx, fmt.Sprintf("Failed to load project %q: %v", project, err))
		return
	}
	_ = p.Reply(context.Background(), msg.ReplyCtx, fmt.Sprintf("Switched to project %s.", project))
}
