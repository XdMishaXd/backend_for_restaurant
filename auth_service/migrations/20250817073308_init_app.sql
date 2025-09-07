-- +goose Up
-- +goose StatementBegin
INSERT INTO apps (id, name, secret)
VALUES (2, 'restaurant', 'pokwdjgo;12!j3ofihwq9587215o&_@$)!*@$&oaqsdfoi^h1324')
ON CONFLICT DO NOTHING;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 'down SQL query';
-- +goose StatementEnd
