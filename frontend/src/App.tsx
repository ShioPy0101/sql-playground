import { AdminPage } from "./pages/AdminPage";
import { PlaygroundPage } from "./pages/PlaygroundPage";
import { TaskListPage } from "./pages/TaskListPage";
import { TaskPage } from "./pages/TaskPage";
import { EventPage } from "./pages/EventPage";
import { AdminEventPage } from "./pages/AdminEventPage";

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

  const adminEventMatch = window.location.pathname.match(/^\/admin\/events\/([^/]+)\/?$/);
  if (adminEventMatch) {
    return <AdminEventPage slug={decodeURIComponent(adminEventMatch[1])} />;
  }

  const eventTaskMatch = window.location.pathname.match(/^\/events\/([^/]+)\/tasks\/(\d+)\/?$/);
  if (eventTaskMatch) {
    return <TaskPage eventSlug={decodeURIComponent(eventTaskMatch[1])} number={eventTaskMatch[2]} />;
  }

  const eventMatch = window.location.pathname.match(/^\/events\/([^/]+)\/?$/);
  if (eventMatch) {
    return <EventPage slug={decodeURIComponent(eventMatch[1])} />;
  }

  const taskMatch = window.location.pathname.match(/^\/tasks\/(\d+)\/?$/);
  if (taskMatch) {
    return <TaskPage number={taskMatch[1]} />;
  }

  return <TaskListPage />;
}

export default App;
