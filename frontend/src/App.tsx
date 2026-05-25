import { AdminPage } from "./pages/AdminPage";
import { PlaygroundPage } from "./pages/PlaygroundPage";
import { TaskListPage } from "./pages/TaskListPage";
import { TaskPage } from "./pages/TaskPage";

function App() {
  if (window.location.pathname.match(/^\/?$/)) {
    return <TaskListPage />;
  }

  if (window.location.pathname.match(/^\/sqlite\/?$/)) {
    return <PlaygroundPage />;
  }

  if (window.location.pathname.match(/^\/admin\/?$/)) {
    return <AdminPage />;
  }

  const taskMatch = window.location.pathname.match(/^\/tasks\/(\d+)\/?$/);
  if (taskMatch) {
    return <TaskPage number={taskMatch[1]} />;
  }

  return <TaskListPage />;
}

export default App;
