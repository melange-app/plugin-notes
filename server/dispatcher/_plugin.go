package dispatcher

import (
	"airdispat.ch/identity"
	"airdispat.ch/message"
	"airdispat.ch/routing"
	"fmt"
	"github.com/airdispatch/dpl"
	"github.com/huntaub/go-cache"
	"log"
	"net/url"
	"time"
)

var profileCache *cache.Cache

type MelangeMessage interface {
	dpl.Message
	// DisplayNews() template.HTML
	// Tags() []dpl.Tag
}

type MessageList []MelangeMessage

func (m MessageList) Len() int               { return len(m) }
func (m MessageList) Less(i int, j int) bool { return m[i].Created().After(m[j].Created()) }
func (m MessageList) Swap(i int, j int)      { m[i], m[j] = m[j], m[i] }

type PluginMail struct {
	*message.Mail
	Profile  *message.Mail
	TProfile *message.Mail
}

func CreatePluginMail(r routing.Router, m *message.Mail, checking *identity.Identity, public bool) *PluginMail {
	if profileCache == nil {
		profileCache = cache.NewCache(time.Hour * 1)
	}
	var profile *message.Mail
	pf, stale := profileCache.Get(m.Header().From.String())
	if !stale {
		profile = pf.(*message.Mail)
	} else {
		pf, err := GetProfile(r, checking, fmt.Sprintf("/%v", m.Header().From.String()))
		if err != nil {
			log.Println("Got error getting profile", err)
			profile = nil
		} else {
			profile = pf
			profileCache.Store(m.Header().From.String(), profile)
		}
	}

	var to *message.Mail
	if !public {
		p, stale := profileCache.Get(m.Header().To.String())
		if !stale {
			to = p.(*message.Mail)
		} else {
			to, err := GetProfile(r, checking, fmt.Sprintf("/%v", m.Header().To.String()))
			if err != nil {
				log.Println("Go an error getting to profile.", err)
				to = nil
			} else {
				profileCache.Store(m.Header().To.String(), to)
			}
		}
	}

	return &PluginMail{
		Mail:     m,
		Profile:  profile,
		TProfile: to,
	}
}

func (p *PluginMail) Components() []dpl.Component {
	var out []dpl.Component = make([]dpl.Component, len(p.Mail.Components))
	i := 0
	for _, v := range p.Mail.Components {
		out[i] = v
		i++
	}
	return out
}

func (p *PluginMail) Get(field string) ([]byte, error) {
	return p.Mail.Components.GetComponent(field), nil
}

func (p *PluginMail) Has(field string) bool {
	return p.Mail.Components.HasComponent(field)
}

func (p *PluginMail) Created() time.Time {
	return time.Unix(p.Header().Timestamp, 0)
}

func (p *PluginMail) Sender() dpl.User {
	return &PluginUser{
		loaded:  p.Header().From,
		profile: p.Profile,
	}
}

func (p *PluginMail) From() []dpl.User {
	if p.TProfile != nil {
		return []dpl.User{&PluginUser{
			loaded:  p.Header().To,
			profile: p.TProfile,
		}}
	}
	return nil
}

// Abstract Getting User's Profile
type PluginUser struct {
	loaded  *identity.Address
	profile *message.Mail
}

func (p *PluginUser) Name() string {
	profile := p.profile
	if profile != nil {
		if profile.Components.HasComponent("airdispat.ch/profile/name") {
			return profile.Components.GetStringComponent("airdispat.ch/profile/name")
		}
	}
	return p.loaded.String()
}

func (p *PluginUser) DisplayAddress() string {
	return p.loaded.String()
}

func (p *PluginUser) Address() string {
	return p.loaded.String()
}

func (p *PluginUser) Avatar() *url.URL {
	profile := p.profile
	if profile != nil {
		if profile.Components.HasComponent("airdispat.ch/profile/avatar") {
			b := profile.Components.GetStringComponent("airdispat.ch/profile/avatar")
			u, err := url.Parse(b)
			if err == nil {
				return u
			}
		}
	}
	u, _ := url.Parse("http://placehold.it/400x400")
	return u
}

func (p *PluginUser) Profile() *url.URL {
	u, _ := url.Parse(fmt.Sprintf("http://www.airdispatch.me/p/%v", p.loaded.String()))
	return u
}