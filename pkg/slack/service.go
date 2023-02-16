package slack

import (
	"fmt"
	"html"

	"github.com/go-logr/logr"
	"github.com/slack-go/slack"

	slackv1alpha1 "github.com/stakater/slack-operator/api/v1alpha1"
)

const (
	ChannelAlreadyExistsError string = "A channel with the same name already exists"
)

// Service interface
type Service interface {
	CreateChannel(string, bool) (*string, error)
	SetDescription(string, string) (*slack.Channel, error)
	SetTopic(string, string) (*slack.Channel, error)
	RenameChannel(string, string) (*slack.Channel, error)
	ArchiveChannel(string) error
	InviteUsers(string, []string) []error
	RemoveUsers(string, []string) error
	GetChannel(string) (*slack.Channel, error)
	GetUsersInChannel(channelID string) ([]string, error)
	GetChannelCRFromChannel(*slack.Channel) *slackv1alpha1.Channel
	IsChannelUpdated(*slackv1alpha1.Channel) (bool, error)
	IsValidChannel(*slackv1alpha1.Channel) error
	GetChannelByName(string) (*slack.Channel, error)
	UnArchiveChannel(*slack.Channel) error
}

// SlackService structure
type SlackService struct {
	log logr.Logger
	api *slack.Client
}

// New creates a new SlackService
func New(APIToken string, logger logr.Logger) *SlackService {
	return &SlackService{
		api: slack.New(APIToken),
		log: logger,
	}
}

// GetChannel gets a channel on slack
func (s *SlackService) GetChannel(channelID string) (*slack.Channel, error) {
	log := s.log.WithValues("channelID", channelID)

	channel, err := s.api.GetConversationInfo(channelID, false)
	if err != nil {
		log.Error(err, "Error fetching channel")
		return nil, err
	}

	return channel, err
}

// CreateChannel creates a public or private channel on slack with the given name
func (s *SlackService) CreateChannel(name string, isPrivate bool) (*string, error) {
	s.log.Info("Creating Slack Channel", "name", name, "isPrivate", isPrivate)

	channel, err := s.api.CreateConversation(name, isPrivate)
	if err != nil {
		return nil, err
	}

	s.log.V(1).Info("Created Slack Channel", "channel", channel)

	return &channel.ID, nil
}

// SetDescription sets description/"purpose" of the slack channel
func (s *SlackService) SetDescription(channelID string, description string) (*slack.Channel, error) {
	log := s.log.WithValues("channelID", channelID)

	channel, err := s.api.GetConversationInfo(channelID, false)

	if err != nil {
		log.Error(err, "Error fetching channel")
		return nil, err
	}

	if html.UnescapeString(channel.Purpose.Value) == description {
		return channel, nil
	}

	log.V(1).Info("Setting Description of the Slack Channel")

	channel, err = s.api.SetPurposeOfConversation(channelID, description)

	if err != nil {
		log.Error(err, "Error setting description of the channel")
		return nil, err
	}
	return channel, nil
}

// SetTopic sets "topic" of the slack channel
func (s *SlackService) SetTopic(channelID string, topic string) (*slack.Channel, error) {
	log := s.log.WithValues("channelID", channelID)

	channel, err := s.api.GetConversationInfo(channelID, false)

	if err != nil {
		log.Error(err, "Error fetching channel")
		return nil, err
	}

	if html.UnescapeString(channel.Topic.Value) == topic {
		return channel, nil
	}

	log.V(1).Info("Setting Topic of the Slack Channel")

	channel, err = s.api.SetTopicOfConversation(channelID, topic)

	if err != nil {
		log.Error(err, "Error setting topic of the channel")
		return nil, err
	}
	return channel, nil
}

// RenameChannel renames the slack channel
func (s *SlackService) RenameChannel(channelID string, newName string) (*slack.Channel, error) {
	log := s.log.WithValues("channelID", channelID)

	channel, err := s.api.GetConversationInfo(channelID, false)

	if err != nil {
		log.Error(err, "Error fetching channel")
		return nil, err
	}
	if html.UnescapeString(channel.Name) == newName {
		return channel, nil
	}

	log.V(1).Info("Renaming Slack Channel", "newName", newName)

	channel, err = s.api.RenameConversation(channelID, newName)

	if err != nil {
		log.Error(err, "Error renaming channel")
		return nil, err
	}
	return channel, nil
}

// ArchiveChannel archives the slack channel
func (s *SlackService) ArchiveChannel(channelID string) error {
	log := s.log.WithValues("channelID", channelID)

	log.V(1).Info("Archiving channel")
	err := s.api.ArchiveConversation(channelID)

	if err != nil {
		log.Error(err, "Error archiving channel")
		return err
	}

	return nil
}

// GetUsersInChannel get all the users in the slack channel
func (s *SlackService) GetUsersInChannel(channelID string) ([]string, error) {
	userIDs, _, err := s.api.GetUsersInConversation(&slack.GetUsersInConversationParameters{
		ChannelID: channelID,
		Limit:     100000,
	})

	return userIDs, err
}

// InviteUsers invites users to the slack channel
func (s *SlackService) InviteUsers(channelID string, userEmails []string) []error {
	log := s.log.WithValues("channelID", channelID)

	var errorlist []error

	for _, email := range userEmails {
		user, err := s.api.GetUserByEmail(email)

		if err != nil {
			errorlist = append(errorlist, fmt.Errorf(fmt.Sprintf("Error fetching user by Email %s", email)))
			continue
		}

		log.V(1).Info("Inviting user to Slack Channel", "userID", user.ID)
		_, err = s.api.InviteUsersToConversation(channelID, user.ID)

		if err != nil && err.Error() != "already_in_channel" && err.Error() != "cant_invite_self" {
			log.Error(err, "Error Inviting user to channel", "userID", user.ID)
			errorlist = append(errorlist, err)
		}
	}

	return errorlist
}

// RemoveUsers remove users from the slack channel
func (s *SlackService) RemoveUsers(channelID string, userEmails []string) error {
	log := s.log.WithValues("channelID", channelID)

	channelUserIDs, err := s.GetUsersInChannel(channelID)
	if err != nil {
		log.Error(err, "Error getting users in a conversation")
		return err
	}

	for _, userId := range channelUserIDs {
		user, err := s.api.GetUserInfo(userId)
		if err != nil {
			log.Error(err, "Error fetching user info")
			return err
		}

		if !user.IsBot {
			found := false
			for _, email := range userEmails {
				if email == user.Profile.Email {
					found = true
					break
				}
			}

			if !found {
				err = s.api.KickUserFromConversation(channelID, user.ID)
				if err != nil {
					log.Error(err, "Error removing user from the conversation")
					return err
				}
			}
		}
	}

	return nil
}

func (s *SlackService) GetChannelCRFromChannel(existingChannel *slack.Channel) *slackv1alpha1.Channel {
	var channel slackv1alpha1.Channel

	channel.Spec.Name = existingChannel.Name
	channel.Spec.Description = existingChannel.Purpose.Value
	channel.Spec.Topic = existingChannel.Topic.Value
	channel.Spec.Private = existingChannel.IsPrivate
	channel.Spec.Users = existingChannel.Members

	return &channel
}

func (s *SlackService) IsChannelUpdated(channel *slackv1alpha1.Channel) (bool, error) {
	log := s.log.WithValues("channelID", channel.Status.ID)

	channelID := channel.Status.ID
	name := channel.Spec.Name
	topic := channel.Spec.Topic
	description := channel.Spec.Description
	userEmails := channel.Spec.Users

	existingChannel, err := s.api.GetConversationInfo(channel.Status.ID, false)
	if err != nil {
		log.Error(err, "Error fetching channel")
		return false, err
	}

	if html.UnescapeString(existingChannel.Name) != name {
		return true, nil
	}
	if html.UnescapeString(existingChannel.Topic.Value) != topic {
		return true, nil
	}
	if html.UnescapeString(existingChannel.Purpose.Value) != description {
		return true, nil
	}

	channelUserIDs, err := s.GetUsersInChannel(channelID)
	if err != nil {
		log.Error(err, "Error getting users in a conversation")
		return false, err
	}

	// Checking if the user is added
	for _, email := range userEmails {
		user, err := s.api.GetUserByEmail(email)
		if err != nil {
			log.Error(err, fmt.Sprintf("Error fetching user by Email %s", email))
			return false, err
		}

		found := false
		for _, id := range channelUserIDs {
			if user.ID == id {
				found = true
				break
			}
		}

		if !found {
			return true, nil
		}
	}

	// Checking if the user is removed
	for _, userId := range channelUserIDs {
		user, err := s.api.GetUserInfo(userId)
		if err != nil {
			log.Error(err, "Error fetching user info")
			return false, err
		}

		if !user.IsBot {
			found := false
			for _, email := range userEmails {
				if email == user.Profile.Email {
					found = true
					break
				}
			}

			if !found {
				return true, nil
			}
		}
	}

	return false, nil
}

func (s *SlackService) IsValidChannel(channel *slackv1alpha1.Channel) error {
	if len(channel.Spec.Users) < 1 {
		return fmt.Errorf("Users can not be empty")
	}

	return nil
}

// GetChannelByName search for the channel on slack by name
func (s *SlackService) GetChannelByName(name string) (*slack.Channel, error) {
	var cursor string

	for {
		channels, nextCursor, err := s.api.GetConversations(&slack.GetConversationsParameters{
			Types: []string{
				"private_channel",
				"public_channel",
			},
			Cursor:          cursor,
			Limit:           200,
			ExcludeArchived: "false",
		})
		if err != nil {
			return nil, err
		}

		for _, channel := range channels {
			if channel.Name == name {
				return &channel, nil
			}
		}

		if nextCursor == "" {
			break
		}
		cursor = nextCursor
	}

	return nil, fmt.Errorf(ChannelAlreadyExistsError)
}

// UnArchiveChannel unarchives the channel
func (s *SlackService) UnArchiveChannel(channel *slack.Channel) error {
	err := s.api.UnArchiveConversation(channel.ID)
	if err != nil {
		return err
	}
	return nil
}
