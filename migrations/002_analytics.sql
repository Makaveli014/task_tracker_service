-- (а) JOIN 3+ таблиц + агрегация
-- "Для каждой команды: название, кол-во участников, кол-во задач done за последние 7 дней"
-- Используется как prepared query в analytics repository
/*
SELECT
    t.name AS team_name,
    COUNT(DISTINCT tm.user_id) AS member_count,
    COUNT(DISTINCT CASE
        WHEN tk.status = 'done' AND tk.updated_at >= DATE_SUB(NOW(), INTERVAL 7 DAY)
        THEN tk.id
    END) AS done_tasks_week
FROM teams t
LEFT JOIN team_members tm ON tm.team_id = t.id
LEFT JOIN tasks tk ON tk.team_id = t.id
GROUP BY t.id, t.name;
*/

-- (б) Оконная функция — топ-3 по созданным задачам в каждой команде за месяц
/*
WITH ranked AS (
    SELECT
        tk.team_id,
        t.name AS team_name,
        u.username,
        COUNT(*) AS task_count,
        ROW_NUMBER() OVER (PARTITION BY tk.team_id ORDER BY COUNT(*) DESC) AS rn
    FROM tasks tk
    JOIN teams t ON t.id = tk.team_id
    JOIN users u ON u.id = tk.created_by
    WHERE tk.created_at >= DATE_SUB(NOW(), INTERVAL 1 MONTH)
    GROUP BY tk.team_id, t.name, u.id, u.username
)
SELECT team_name, username, task_count
FROM ranked
WHERE rn <= 3
ORDER BY team_name, task_count DESC;
*/

-- (в) Валидация целостности: assignee не является членом команды задачи
/*
SELECT tk.id, tk.title, tk.assignee_id, tk.team_id, u.username AS assignee_name
FROM tasks tk
JOIN users u ON u.id = tk.assignee_id
LEFT JOIN team_members tm ON tm.team_id = tk.team_id AND tm.user_id = tk.assignee_id
WHERE tk.assignee_id IS NOT NULL AND tm.id IS NULL;
*/
