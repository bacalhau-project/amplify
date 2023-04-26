import Button from '@mui/material/Button';
import Card from '@mui/material/Card';
import CardActions from '@mui/material/CardActions';
import CardContent from '@mui/material/CardContent';
import Grid from '@mui/material/Grid';
import Typography from '@mui/material/Typography';
import { Datagrid, List, NumberField, Resource, TextField, Title } from 'react-admin';

export default () => (
    <Grid container spacing={2}>

        <Grid item xs={12}>
            <Card sx={{ minWidth: 275 }}>
                <CardContent>
                    <h1>
                        <Typography variant="h3" component="div">
                            Bacalhau Amplify
                        </Typography>
                    </h1>
                    <Typography variant="body2">
                        Bacalhau Amplify is a decentralized, open-source, and community-driven project to automatically enrich, enhance, and explain data.
                        <br />
                        <br />
                        This is the administrative interface for the Bacalhau Amplify project.
                    </Typography>
                </CardContent>
                <CardActions>
                    <a href="https://github.com/bacalhau-project/amplify/">
                        <Button size="small">Learn More</Button>
                    </a>
                </CardActions>
            </Card>
        </Grid>
        <Grid item sm={12} md={6} lg={4}>
            <Card>
                <CardContent>
                    <h3>
                        <Typography variant="h5" >
                            Top 10 Content-Type
                        </Typography>
                    </h3>
                    <Typography variant="body2">
                        This table shows the top 10 mime-types of all files flowing through Amplify. This data is produced by the metadata-job and stored in the database.
                    </Typography>
                    <Resource name="analytics/results/content-type" list={ResultList} hasEdit={false} hasShow={false} hasCreate={false} options={{ label: 'Content-Type' }} />
                </CardContent>
            </Card>
        </Grid>
    </Grid>
);

const ResultList = () => (
    <List pagination={false} bulkActionButtons={false} actions={false}>
        <Datagrid rowClick={false} bulkActionButtons={false} >
            <TextField source="id" label="Content-Type" sortable={false} />
            <NumberField source="meta.count" label="Count" sortable={false} />
        </Datagrid>
    </List>
);

// const ContentTypeBarChart = ({ }) => {
//     const { data, total, isLoading, error, refetch } = useGetList(
//         'analytics/results/content-type',
//         { pagination: { perPage: 10, page: 1 } },
//     );

//     if (!data) return null;


//     let plotData = data.map((item: any) => ({
//         "total": item.meta.count,
//         "group": item.id,
//     }));

//     return (
//         <ResponsiveBar
//             data={plotData}
//             keys={['total']}
//             indexBy="group"
//             layers={['grid', 'axes', 'bars', 'markers', 'legends']}
//             margin={{ top: 50, right: 130, bottom: 50, left: 60 }}
//             padding={0.05}
//         />
//     );
// };
