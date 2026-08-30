package broadcast

import (
	"fmt"
	"html"
	"regexp"
	"strings"
)

var broadcastPlaceholderPattern = regexp.MustCompile(`\{\{\{\s*([A-Za-z][A-Za-z0-9_]*)\s*\}\}\}`)

// RenderBroadcast renders a broadcast-owned message with the supplied values.
// Broadcast variable bindings provide defaults; supplied values win so fanout
// can overlay recipient data and managed variables such as unsubscribe links.
func RenderBroadcast(value Broadcast, supplied map[string]any) (PreviewResponse, error) {
	bindings := mergeBindings(value.VariableBindings, supplied)
	return renderContent(
		optionalString(value.FromEmail), value.FromName, value.ReplyToEmail,
		value.Subject, value.PreviewText, value.HTML, value.Text, bindings,
	)
}

// RenderFanoutRecipient renders the immutable message snapshot claimed together
// with a recipient. It deliberately performs no template lookup.
func RenderFanoutRecipient(recipient FanoutRecipient, supplied map[string]any) (PreviewResponse, error) {
	bindings := mergeBindings(recipient.VariableBindings, supplied)
	return renderContent(
		recipient.FromEmail, recipient.FromName, recipient.ReplyToEmail,
		recipient.Subject, recipient.PreviewText, recipient.HTML, recipient.Text, bindings,
	)
}

func renderContent(
	fromEmail, fromName, replyTo *string,
	subject string,
	previewText *string,
	htmlBody string,
	textBody *string,
	bindings map[string]any,
) (PreviewResponse, error) {
	renderedSubject, err := renderString(subject, bindings, false)
	if err != nil {
		return PreviewResponse{}, err
	}
	renderedHTML, err := renderString(htmlBody, bindings, true)
	if err != nil {
		return PreviewResponse{}, err
	}
	renderedPreview, err := renderOptionalString(previewText, bindings, false)
	if err != nil {
		return PreviewResponse{}, err
	}
	renderedText, err := renderOptionalString(textBody, bindings, false)
	if err != nil {
		return PreviewResponse{}, err
	}
	return PreviewResponse{
		FromEmail: optionalStringValue(fromEmail),
		FromName: fromName,
		ReplyToEmail: replyTo,
		Subject: renderedSubject,
		PreviewText: renderedPreview,
		HTML: renderedHTML,
		Text: renderedText,
	}, nil
}

func mergeBindings(base, supplied map[string]any) map[string]any {
	values := make(map[string]any, len(base)+len(supplied))
	for key, value := range base {
		values[key] = value
	}
	for key, value := range supplied {
		values[key] = value
	}
	return values
}

func renderString(input string, values map[string]any, escape bool) (string, error) {
	var renderErr error
	output := broadcastPlaceholderPattern.ReplaceAllStringFunc(input, func(match string) string {
		parts := broadcastPlaceholderPattern.FindStringSubmatch(match)
		key := strings.TrimSpace(parts[1])
		value, ok := values[key]
		if !ok || value == nil {
			renderErr = fmt.Errorf("broadcast variable %s is required", key)
			return match
		}
		text := fmt.Sprint(value)
		if escape {
			return html.EscapeString(text)
		}
		return text
	})
	if renderErr != nil {
		return "", renderErr
	}
	return output, nil
}

func renderOptionalString(input *string, values map[string]any, escape bool) (*string, error) {
	if input == nil {
		return nil, nil
	}
	value, err := renderString(*input, values, escape)
	if err != nil {
		return nil, err
	}
	return &value, nil
}

func optionalString(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	copy := value
	return &copy
}
