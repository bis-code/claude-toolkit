mod lib;
use lib::{process, Task};

#[tokio::main]
async fn main() {
    let task = Task {
        id: 1,
        payload: "process-order-42".to_string(),
    };

    let result = process(task);
    println!(
        "Task {} completed: success={}",
        result.task_id, result.success
    );
}
