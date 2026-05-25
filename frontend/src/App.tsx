import { PlaygroundPage } from "./pages/PlaygroundPage";
import { TaskPage } from "./pages/TaskPage";

function App() {
  const taskMatch = window.location.pathname.match(/^\/tasks\/(\d+)\/?$/);
  if (taskMatch) {
    return <TaskPage number={taskMatch[1]} />;
  }

  return <PlaygroundPage />;
}

export default App;
