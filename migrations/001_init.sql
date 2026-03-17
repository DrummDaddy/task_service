CREATE TABLE IF NOT EXISTS users (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    email VARCHAR(255) NOT NULL,
    password_hash VARBINARY(255) NOT NULL,
    created_at TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    UNIQUE KEY ux_users_email (email)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS teams (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    name VARCHAR(200) NOT NULL,
    created_by BIGINT UNSIGNED NOT NULL,
    created_at TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY(id),
    KEY ix_teams_created_by (created_by),
    CONSTRAINT fk_teams_created_by FOREIGN KEY (created_by) REFERENCES user(id)
        ON DELETE RESTRICT ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS teams (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    name VARCHAR(200) NOT NULL,
    created_by BIGINT UNSIGNED NOT NULL,
    created_at TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    KEY ix_teams_created_by (created_by),
    CONSTRAINT fk_teams_created_by FOREIGN KEY (created_by) REFERENCES users(id)
    ON DELETE RESTRICT ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS tasks (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    team_id BIGINT UNSIGNED NOT NULL,
    title VARCHAR(300) NOT NULL,
    description TEXT NULL,
    status ENUM('todo', 'in_progress', 'done') NOT NULL DEFAULT 'todo',
    assignee_id BIGINT UNSIGNED NULL,
    created_by BIGINT UNSIGNED NOT NULL,
    created_at TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    KEY ix_tasks_team_status_updated (team_id, status, updated_at),
    KEY ix_tasks_team_assignee_status (team_id, assignee_id, status),
    KEY ix_tasks_created_by (created_by),
    CONSTRAINT fk_tasks_team FOREIGN KEY (team_id) REFERENCES teams(id)
    ON DELETE CASCADE ON UPDATE CASCADE,
    CONSTRAINT fk_tasks_assignee FOREIGN KEY (assignee_id) REFERENCES users(id)
    ON DELETE SET NULL ON UPDATE CASCADE,
    CONSTRAINT fk_tasks_created_by FOREIGN KEY (created_by) REFERENCES users(id)
    ON DELETE RESTRICT ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS task_history (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    task_id BIGINT UNSIGNED NOT NULL,
    changed_by BIGINT UNSIGNED NOT NULL,
    field_name VARCHAR(64) NOT NULL,
    old_value TEXT NULL,
    new_value TEXT NULL,
    changed_at TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    KEY ix_task_history_task_changed_at (task_id, changed_at),
    KEY ix_task_history_changed_by (changed_by),
    CONSTRAINT fk_task_history_task FOREIGN KEY (task_id) REFERENCES tasks(id)
    ON DELETE CASCADE ON UPDATE CASCADE,
    CONSTRAINT fk_task_history_changed_by FOREIGN KEY (changed_by) REFERENCES users(id)
    ON DELETE RESTRICT ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS task_comments (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    task_id BIGINT UNSIGNED NOT NULL,
    user_id BIGINT UNSIGNED NOT NULL,
    body TEXT NOT NULL,
    created_at TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    KEY ix_task_comments_task_created_at (task_id, created_at),
    KEY ix_task_comments_user (user_id),
    CONSTRAINT fk_task_comments_task FOREIGN KEY (task_id) REFERENCES tasks(id)
    ON DELETE CASCADE ON UPDATE CASCADE,
    CONSTRAINT fk_task_comments_user FOREIGN KEY (user_id) REFERENCES users(id)
    ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4_unicode_ci;