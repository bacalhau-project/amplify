import AccountTreeIcon from '@mui/icons-material/AccountTree';
import FindInPageIcon from '@mui/icons-material/FindInPage';
import ImageSearchIcon from '@mui/icons-material/ImageSearch';
import ShortTextIcon from '@mui/icons-material/ShortText';
import WorkIcon from '@mui/icons-material/Work';
import { Admin, Resource, ShowGuesser, defaultTheme } from "react-admin";
import { ContentList, RecentResultList } from './Analytics';
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
    <Resource name="analytics/results/content-type" list={ContentList} hasShow={false} hasCreate={false} hasEdit={false} options={{ label: 'Analytics: Content Types' }} icon={FindInPageIcon} />
    <Resource name="analytics/results/content-classification" list={ContentList} hasShow={false} hasCreate={false} hasEdit={false} options={{ label: 'Analytics: Content Classifications' }} icon={ImageSearchIcon} />
    <Resource name="analytics/recent-results/summary_text" list={RecentResultList} hasShow={false} hasCreate={false} hasEdit={false} options={{ label: 'Analytics: Text Summaries' }} icon={ShortTextIcon} />
  </Admin >
);
export default App;
