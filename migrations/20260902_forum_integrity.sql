-- Review before running. This migration is intentionally not executed by
-- AutoMigrate because it deduplicates live interaction rows and recalculates
-- cached counters. Take a database backup first.
BEGIN;

-- Keep the earliest active toggle and soft-delete later duplicates.
WITH duplicates AS (
  SELECT id, ROW_NUMBER() OVER (PARTITION BY user_id, post_id ORDER BY id) AS position
  FROM pkuphysu_forum_follows WHERE deleted_at IS NULL
)
UPDATE pkuphysu_forum_follows SET deleted_at = NOW()
WHERE id IN (SELECT id FROM duplicates WHERE position > 1);

WITH duplicates AS (
  SELECT id, ROW_NUMBER() OVER (PARTITION BY user_id, post_id ORDER BY id) AS position
  FROM pkuphysu_forum_likes WHERE deleted_at IS NULL
)
UPDATE pkuphysu_forum_likes SET deleted_at = NOW()
WHERE id IN (SELECT id FROM duplicates WHERE position > 1);

WITH duplicates AS (
  SELECT id, ROW_NUMBER() OVER (PARTITION BY user_id, comment_id ORDER BY id) AS position
  FROM pkuphysu_comment_likes WHERE deleted_at IS NULL
)
UPDATE pkuphysu_comment_likes SET deleted_at = NOW()
WHERE id IN (SELECT id FROM duplicates WHERE position > 1);

CREATE UNIQUE INDEX IF NOT EXISTS uq_forum_follows_active
  ON pkuphysu_forum_follows (user_id, post_id) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS uq_forum_likes_active
  ON pkuphysu_forum_likes (user_id, post_id) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS uq_comment_likes_active
  ON pkuphysu_comment_likes (user_id, comment_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS ix_forum_posts_feed ON pkuphysu_forum_posts (deleted_at, id DESC);
CREATE INDEX IF NOT EXISTS ix_forum_comments_post ON pkuphysu_forum_comments (post_id, deleted_at, id);

UPDATE pkuphysu_forum_posts post SET
  follownum = (SELECT COUNT(*) FROM pkuphysu_forum_follows item WHERE item.post_id = post.id AND item.deleted_at IS NULL),
  likenum = (SELECT COUNT(*) FROM pkuphysu_forum_likes item WHERE item.post_id = post.id AND item.deleted_at IS NULL),
  reply = (SELECT COUNT(*) FROM pkuphysu_forum_comments item WHERE item.post_id = post.id AND item.deleted_at IS NULL);

UPDATE pkuphysu_forum_comments comment SET
  likenum = (SELECT COUNT(*) FROM pkuphysu_comment_likes item WHERE item.comment_id = comment.id AND item.deleted_at IS NULL);

COMMIT;
