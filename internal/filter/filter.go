package filter

import (
	"bufio"
	"os"
	"strings"
	"sync"
)

type Filter struct {
	blocked map[string]bool
	enabled bool
	mu      sync.RWMutex
}

func New(path string, enabled bool) (*Filter, error) {
	f := &Filter{
		blocked: make(map[string]bool),
		enabled: enabled,
	}

	if enabled && path != "" {
		if err := f.LoadFile(path); err != nil {
			return nil, err
		}
	}

	return f, nil
}

func (f *Filter) LoadFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	f.mu.Lock()
	defer f.mu.Unlock()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) >= 2 {
			domain := strings.ToLower(parts[1])
			f.blocked[domain] = true
		}
	}

	return scanner.Err()
}

func (f *Filter) IsBlocked(domain string) bool {
	f.mu.RLock()
	defer f.mu.RUnlock()

	domain = strings.ToLower(strings.TrimSuffix(domain, "."))

	for {
		if f.blocked[domain] {
			return true
		}

		dotIdx := strings.Index(domain, ".")
		if dotIdx == -1 {
			break
		}
		domain = domain[dotIdx+1:]
	}

	return false
}

func (f *Filter) IsEnabled() bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.enabled
}

func (f *Filter) SetEnabled(enabled bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.enabled = enabled
}

func (f *Filter) AddDomain(domain string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.blocked[strings.ToLower(domain)] = true
}

func (f *Filter) RemoveDomain(domain string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.blocked, strings.ToLower(domain))
	return true
}

func (f *Filter) Domains() []string {
	f.mu.RLock()
	defer f.mu.RUnlock()

	domains := make([]string, 0, len(f.blocked))
	for d := range f.blocked {
		domains = append(domains, d)
	}
	return domains
}

func (f *Filter) Count() int {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return len(f.blocked)
}
