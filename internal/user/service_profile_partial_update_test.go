package user

import (
	"testing"
	"time"

	"github.com/maintainerd/maintainerd-auth/internal/platform/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
)

// Every optional profile field arrives as a pointer that decodes to nil when the
// JSON key is absent, and the row is persisted with a full-struct Save. The
// assignments used to be unconditional, so a client that PUT only
// {first_name, last_name} silently erased gender, middle name, birthdate and the
// whole metadata blob — which carries the OIDC `address` claim. A partial update
// must not be a destructive one.
func TestApplyProfileFields_OmittedFieldsAreNotWiped(t *testing.T) {
	birthdate := time.Date(1990, 1, 25, 0, 0, 0, 0, time.UTC)
	existing := Profile{
		FirstName:   "Alice",
		MiddleName:  ptr.Ptr("Q"),
		LastName:    ptr.Ptr("Smith"),
		DisplayName: ptr.Ptr("Ali"),
		Birthdate:   &birthdate,
		Gender:      ptr.Ptr("female"),
		Timezone:    ptr.Ptr("UTC"),
		Language:    ptr.Ptr("en"),
		ProfileURL:  ptr.Ptr("https://example.com/a.png"),
	}

	t.Run("absent fields are preserved", func(t *testing.T) {
		p := existing
		// Only first_name and last_name were sent.
		applyProfileFields(&p, "Alicia", nil, ptr.Ptr("Jones"), nil, nil, nil, nil, nil, nil)

		assert.Equal(t, "Alicia", p.FirstName)
		assert.Equal(t, "Jones", *p.LastName)
		require.NotNil(t, p.MiddleName)
		assert.Equal(t, "Q", *p.MiddleName)
		require.NotNil(t, p.DisplayName)
		assert.Equal(t, "Ali", *p.DisplayName)
		require.NotNil(t, p.Birthdate)
		assert.Equal(t, birthdate, *p.Birthdate)
		require.NotNil(t, p.Gender)
		assert.Equal(t, "female", *p.Gender)
		require.NotNil(t, p.Timezone)
		assert.Equal(t, "UTC", *p.Timezone)
		require.NotNil(t, p.Language)
		assert.Equal(t, "en", *p.Language)
		require.NotNil(t, p.ProfileURL)
		assert.Equal(t, "https://example.com/a.png", *p.ProfileURL)
	})

	t.Run("sent fields are applied", func(t *testing.T) {
		p := existing
		newBirthdate := time.Date(1991, 2, 3, 0, 0, 0, 0, time.UTC)
		applyProfileFields(&p, "Alice", ptr.Ptr("R"), ptr.Ptr("Brown"), ptr.Ptr("Al"),
			&newBirthdate, ptr.Ptr("other"), ptr.Ptr("Europe/Berlin"), ptr.Ptr("de"),
			ptr.Ptr("https://example.com/b.png"))

		assert.Equal(t, "R", *p.MiddleName)
		assert.Equal(t, "Brown", *p.LastName)
		assert.Equal(t, "Al", *p.DisplayName)
		assert.Equal(t, newBirthdate, *p.Birthdate)
		assert.Equal(t, "other", *p.Gender)
		assert.Equal(t, "Europe/Berlin", *p.Timezone)
		assert.Equal(t, "de", *p.Language)
		assert.Equal(t, "https://example.com/b.png", *p.ProfileURL)
	})
}

func TestApplyProfileMetadata(t *testing.T) {
	t.Run("omitted metadata leaves the stored blob intact", func(t *testing.T) {
		// The OIDC `address` claim lives in here; an update that does not mention
		// metadata must not drop it.
		p := Profile{Metadata: datatypes.JSON([]byte(`{"address":{"country":"PH"}}`))}

		require.NoError(t, applyProfileMetadata(&p, nil))

		assert.JSONEq(t, `{"address":{"country":"PH"}}`, string(p.Metadata))
	})

	t.Run("sent metadata replaces the blob", func(t *testing.T) {
		p := Profile{Metadata: datatypes.JSON([]byte(`{"address":{"country":"PH"}}`))}

		require.NoError(t, applyProfileMetadata(&p, map[string]any{"nickname": "Ali"}))

		assert.JSONEq(t, `{"nickname":"Ali"}`, string(p.Metadata))
	})

	t.Run("a new row is seeded with an empty object", func(t *testing.T) {
		p := Profile{}

		require.NoError(t, applyProfileMetadata(&p, nil))

		assert.JSONEq(t, `{}`, string(p.Metadata))
	})
}

// The request DTO declares birthdate as "YYYY-MM-DD" but the response used to
// serialize a *time.Time as RFC3339, so echoing a GET back on a PUT failed
// validateDateFormat — the read and write halves of one field disagreed.
func TestProfileBirthdateRoundTrips(t *testing.T) {
	stored := time.Date(1990, 1, 25, 0, 0, 0, 0, time.UTC)

	out := toProfileResponseDTO(ProfileServiceDataResult{FirstName: "Alice", Birthdate: &stored})

	require.NotNil(t, out.Birthdate)
	assert.Equal(t, "1990-01-25", *out.Birthdate)

	// Feeding the response value straight back into a request must validate.
	back := ProfileRequestDTO{FirstName: "Alice", Birthdate: out.Birthdate}
	assert.NoError(t, back.Validate())
}

func TestProfileBirthdateNilStaysNil(t *testing.T) {
	out := toProfileResponseDTO(ProfileServiceDataResult{FirstName: "Alice"})
	assert.Nil(t, out.Birthdate)
}
