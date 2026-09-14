package image

// imageColumns is the column list every query below returns, in this order;
// postgres.go's scan calls depend on it.
const imageColumns = `id, product_id, object_key, alt, position, created_at`

// insertImageStmt's position subquery keeps images ordered by upload order
// without the caller having to track a position itself.
const insertImageStmt = `INSERT INTO product_images (product_id, object_key, alt, position)
VALUES ($1, $2, $3, COALESCE((SELECT MAX(position) + 1 FROM product_images WHERE product_id = $1), 0))
RETURNING ` + imageColumns

const getImageQuery = `SELECT ` + imageColumns + ` FROM product_images WHERE product_id = $1 AND id = $2`

const deleteImageStmt = `DELETE FROM product_images WHERE product_id = $1 AND id = $2`

const listImagesByProductQuery = `SELECT ` + imageColumns + ` FROM product_images WHERE product_id = $1 ORDER BY position, id`
