use serde::{Deserialize, Serialize};

#[derive(Debug, Serialize, Deserialize)]
pub struct Task {
    pub id: u32,
    pub payload: String,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct TaskResult {
    pub task_id: u32,
    pub success: bool,
}

pub fn process(task: Task) -> TaskResult {
    TaskResult {
        task_id: task.id,
        success: !task.payload.is_empty(),
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_process_non_empty_payload() {
        let task = Task { id: 1, payload: "work".to_string() };
        let result = process(task);
        assert!(result.success);
        assert_eq!(result.task_id, 1);
    }

    #[test]
    fn test_process_empty_payload_fails() {
        let task = Task { id: 2, payload: String::new() };
        let result = process(task);
        assert!(!result.success);
    }
}
