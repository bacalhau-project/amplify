import AccountTreeIcon from '@mui/icons-material/AccountTree';
import WorkIcon from '@mui/icons-material/Work';
import { Admin, Resource, ShowGuesser, defaultTheme } from "react-admin";
import Dashboard from './Dashboard';
import { GraphList } from './Graph';
import { JobList } from './Jobs';
import { QueueCreate, QueueItem, QueueList } from './Queue';
import { dataProvider } from './dataProvider';

const theme = {
  ...defaultTheme,
  palette: {
    ...defaultTheme.palette,
    background: {
      default: '#FFFFFF',
      paper: '#FFFFFF',
    },
  }
};

const App = () => (
  <Admin
    dataProvider={dataProvider}
    dashboard={Dashboard}
    theme={theme}
    disableTelemetry
  >
    <Resource name="queue" list={QueueList} show={QueueItem} create={QueueCreate} options={{ label: 'Queue' }} />
    <Resource name="graph" list={GraphList} options={{ label: 'Graph' }} icon={AccountTreeIcon} />
    <Resource name="jobs" list={JobList} show={ShowGuesser} options={{ label: 'Jobs' }} icon={WorkIcon} />
  </Admin >
);
export default App;
