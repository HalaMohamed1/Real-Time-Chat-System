package integration

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"rtcs/internal/model"
	"rtcs/internal/repository"
)

func TestRepository_User(t *testing.T) {
	gormDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn: db,
	}), &gorm.Config{})
	require.NoError(t, err)

	userRepo := repository.NewUserRepository(gormDB)
	ctx := context.Background()

	t.Run("Create User", func(t *testing.T) {
		user := &model.User{
			Username:    "repouser",
			Password:    "hashedpassword",
			DisplayName: "Repository Test User",
			AvatarURL:   "https://example.com/avatar.png",
			About:       "Test user for repository tests",
		}

		err := userRepo.Create(ctx, user)
		require.NoError(t, err)

		assert.NotEqual(t, uuid.Nil, user.ID)

		var count int
		err = db.QueryRow(
			"SELECT COUNT(*) FROM users WHERE username = $1",
			"repouser",
		).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 1, count)
	})

	t.Run("Get User By ID", func(t *testing.T) {
		user, err := userRepo.GetByID(ctx, testUserID)
		require.NoError(t, err)

		assert.Equal(t, testUserID, user.ID)
		assert.Equal(t, "testuser", user.Username)
		assert.Equal(t, "Test User", user.DisplayName)
	})
	t.Run("Get User By Username", func(t *testing.T) {
		user, err := userRepo.GetByUsername(ctx, "testuser")
		require.NoError(t, err)
		assert.Equal(t, testUserID, user.ID)
		assert.Equal(t, "testuser", user.Username)
	})

	t.Run("Update User Profile", func(t *testing.T) {
		profile := &model.UserProfile{
			DisplayName: "Updated Repository User",
			AvatarURL:   "https://example.com/newavatar.png",
			About:       "Updated profile for repository tests",
		}

		err := userRepo.UpdateProfile(ctx, testUserID, profile)
		require.NoError(t, err)

		var displayName, avatarURL, about string
		err = db.QueryRow(
			"SELECT display_name, avatar_url, about FROM users WHERE id = $1",
			testUserID,
		).Scan(&displayName, &avatarURL, &about)
		require.NoError(t, err)
		assert.Equal(t, "Updated Repository User", displayName)
		assert.Equal(t, "https://example.com/newavatar.png", avatarURL)
		assert.Equal(t, "Updated profile for repository tests", about)
	})
}

func TestRepository_Chat(t *testing.T) {
	gormDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn: db,
	}), &gorm.Config{})
	require.NoError(t, err)

	chatRepo := repository.NewChatRepository(gormDB)
	ctx := context.Background()

	t.Run("Create Chat", func(t *testing.T) {
		chat := &model.Chat{
			Name: "Repository Test Chat",
		}

		err := chatRepo.CreateChat(ctx, chat)
		require.NoError(t, err)

		assert.NotEqual(t, uuid.Nil, chat.ID)

		var count int
		err = db.QueryRow(
			"SELECT COUNT(*) FROM chats WHERE name = $1",
			"Repository Test Chat",
		).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 1, count)
	})

	t.Run("Get Chat", func(t *testing.T) {
		chat, err := chatRepo.GetChat(ctx, testChatID)
		require.NoError(t, err)

		assert.Equal(t, testChatID, chat.ID)
		assert.Equal(t, "Test Chat", chat.Name)
	})

	t.Run("List Chats", func(t *testing.T) {
		chats, err := chatRepo.ListChats(ctx, testUserID)
		require.NoError(t, err)

		assert.GreaterOrEqual(t, len(chats), 1)

		found := false
		for _, chat := range chats {
			if chat.ID == testChatID {
				found = true
				assert.Equal(t, "Test Chat", chat.Name)
				break
			}
		}
		assert.True(t, found, "Test chat not found in the list")
	})

	t.Run("Add User To Chat", func(t *testing.T) {
		newUserID := uuid.New()
		_, err = db.Exec(
			"INSERT INTO users (id, username, password) VALUES ($1, $2, $3)",
			newUserID, "chatuser", "password",
		)
		require.NoError(t, err)

		err = chatRepo.AddUserToChat(ctx, testChatID, newUserID)
		require.NoError(t, err)

		var count int
		err = db.QueryRow(
			"SELECT COUNT(*) FROM chat_users WHERE chat_id = $1 AND user_id = $2",
			testChatID, newUserID,
		).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 1, count)
	})

	t.Run("Remove User From Chat", func(t *testing.T) {
		var userID uuid.UUID
		err = db.QueryRow(
			"SELECT user_id FROM chat_users WHERE chat_id = $1 AND user_id != $2 LIMIT 1",
			testChatID, testUserID,
		).Scan(&userID)
		require.NoError(t, err)

		err = chatRepo.RemoveUserFromChat(ctx, testChatID, userID)
		require.NoError(t, err)

		var count int
		err = db.QueryRow(
			"SELECT COUNT(*) FROM chat_users WHERE chat_id = $1 AND user_id = $2",
			testChatID, userID,
		).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})
}

func TestRepository_Message(t *testing.T) {
	gormDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn: db,
	}), &gorm.Config{})
	require.NoError(t, err)

	messageRepo := repository.NewMessageRepository(gormDB)
	ctx := context.Background()

	t.Run("Save Message", func(t *testing.T) {
		message := &model.Message{
			ID:        uuid.New(),
			ChatID:    testChatID,
			SenderID:  testUserID,
			Text:      "Repository test message",
			CreatedAt: time.Now(),
		}

		err := messageRepo.SaveMessage(ctx, message)
		require.NoError(t, err)
		var text string
		err = db.QueryRow(
			"SELECT text FROM messages WHERE id = $1",
			message.ID,
		).Scan(&text)
		require.NoError(t, err)
		assert.Equal(t, "Repository test message", text)
	})

	t.Run("Get Messages", func(t *testing.T) {
		for i := 0; i < 3; i++ {
			message := &model.Message{
				ID:        uuid.New(),
				ChatID:    testChatID,
				SenderID:  testUserID,
				Text:      "Test message " + time.Now().String(),
				CreatedAt: time.Now(),
			}
			err := messageRepo.SaveMessage(ctx, message)
			require.NoError(t, err)
		}

		messages, err := messageRepo.GetMessages(ctx, testChatID, 5)
		require.NoError(t, err)

		assert.GreaterOrEqual(t, len(messages), 3)
		for _, msg := range messages {
			assert.Equal(t, testChatID, msg.ChatID)
			assert.Equal(t, testUserID, msg.SenderID)
			assert.NotEmpty(t, msg.Text)
		}
	})

	// Test getting a specific message
	t.Run("Get Message", func(t *testing.T) {
		// Create a message
		messageID := uuid.New()
		message := &model.Message{
			ID:        messageID,
			ChatID:    testChatID,
			SenderID:  testUserID,
			Text:      "Specific test message",
			CreatedAt: time.Now(),
		}
		err := messageRepo.SaveMessage(ctx, message)
		require.NoError(t, err)

		// Get the message
		retrievedMsg, err := messageRepo.GetMessage(ctx, messageID)
		require.NoError(t, err)

		// Verify message data
		assert.Equal(t, messageID, retrievedMsg.ID)
		assert.Equal(t, testChatID, retrievedMsg.ChatID)
		assert.Equal(t, testUserID, retrievedMsg.SenderID)
		assert.Equal(t, "Specific test message", retrievedMsg.Text)
	})

	// Test deleting a message
	t.Run("Delete Message", func(t *testing.T) {
		// Create a message
		messageID := uuid.New()
		message := &model.Message{
			ID:        messageID,
			ChatID:    testChatID,
			SenderID:  testUserID,
			Text:      "Message to delete",
			CreatedAt: time.Now(),
		}
		err := messageRepo.SaveMessage(ctx, message)
		require.NoError(t, err)

		// Delete the message
		err = messageRepo.DeleteMessage(ctx, messageID)
		require.NoError(t, err)

		// Verify in database (soft delete)
		var deletedAt *time.Time
		err = db.QueryRow(
			"SELECT deleted_at FROM messages WHERE id = $1",
			messageID,
		).Scan(&deletedAt)
		require.NoError(t, err)
		assert.NotNil(t, deletedAt)
	})
}
