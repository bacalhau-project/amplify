import Button from '@mui/material/Button';
import Card from '@mui/material/Card';
import CardActions from '@mui/material/CardActions';
import CardContent from '@mui/material/CardContent';
import Grid from '@mui/material/Grid';
import Typography from '@mui/material/Typography';
import { Datagrid, List, NumberField, Resource, TextField, Title, useGetList } from 'react-admin';
import { ResponsiveLine } from '@nivo/line'

export default () => (
    <div>
        <Title title="Bacalhau Amplify" />
        <Grid container spacing={2}>
            <Grid item xs={12}>
                <Card sx={{ minWidth: 275 }}>
                    <CardContent>
                        <Typography variant="h1" component="div">
                            Bacalhau Amplify
                        </Typography>
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
                        <Typography variant="h3" >
                            Top Content-Type
                        </Typography>
                        <Typography variant="body2">
                            This table shows the top mime-types of all files flowing through Amplify. This data is produced by the metadata-job and stored in the database.
                        </Typography>
                        <Resource name="analytics/results/content-type" list={ResultList} hasEdit={false} hasShow={false} hasCreate={false} options={{ label: 'Content-Type' }} />
                    </CardContent>
                </Card>
            </Grid>
            <Grid item sm={12} md={6} lg={4}>
                <Card>
                    <CardContent>
                        <Typography variant="h3" >
                            Top Content-Classification
                        </Typography>
                        <Typography variant="body2">
                            This table shows the top object classifications from all images and videos flowing through Amplify. This data is produced by the detection job and stored in the database.
                        </Typography>
                        <Resource name="analytics/results/content-classification" list={ResultList} hasEdit={false} hasShow={false} hasCreate={false} options={{ label: 'content-classification' }} />
                    </CardContent>
                </Card>
            </Grid>
            <Grid item sm={12} md={6} lg={4}>
                <Card>
                    <CardContent>
                        <Typography variant="h3" >
                            Amplify Metrics
                        </Typography>
                        <Typography variant="body2">
                           This chart shows the number of submissions and node executions over time. These numbers are aggregated based upon the timestamps in the database.
                        </Typography>
                        <ContentTypeBarChart />
                    </CardContent>
                </Card>
            </Grid>

            <Grid item sm={12} md={12} lg={12}>
                <Card>
                    <CardContent>
                        <Typography variant="h3" >
                            Most Recent Summaries
                        </Typography>
                        <Typography variant="body2">
                            This table shows the most recent text summaries of the content flowing through Amplify.
                        </Typography>
                        <Resource name="analytics/recent-results/summary_text" list={RecentResultList} hasEdit={false} hasShow={false} hasCreate={false} options={{ label: 'summary_text' }} />
                    </CardContent>
                </Card>
            </Grid>
        </Grid>
    </div>
);

const ResultList = () => (
    <List pagination={false} bulkActionButtons={false} actions={false} title={<div></div>} sort={{ field: 'meta.count', order: 'DESC' }}>
        <Datagrid rowClick={false} bulkActionButtons={false} >
            <TextField source="id" label="Content-Type" sortable={false} />
            <NumberField source="meta.count" label="Count" />
        </Datagrid>
    </List>
);

const RecentResultList = () => (
    <List pagination={false} bulkActionButtons={false} actions={false} title={<div></div>} sort={{ field: 'meta.created_at', order: 'DESC' }}>
        <Datagrid rowClick={false} bulkActionButtons={false} >
            <NumberField source="meta.created_at" noWrap />
            <TextField source="id" sortable={false} />
        </Datagrid>
    </List>
);

const ContentTypeBarChart = ({ }) => {
    const dataA = useGetList(
        'analytics/metrics-over-time/submissions',
        { pagination: { perPage: 10, page: 1 } },
    );

    const dataB = useGetList(
        'analytics/metrics-over-time/node-results',
        { pagination: { perPage: 10, page: 1 } },
    );

    if (!dataA.data || !dataB.data) return null;

    let plotData = [{
        id: "Submissions",
        data: dataA.data.map((item: any) => ({
        "y": item.meta.count,
        "x": item.id,
    }))
    },{
        id: "Node Executions",
        data: dataB.data.map((item: any) => ({
        "y": item.meta.count,
        "x": item.id,
    }))
    }];

    return (
        <div style={{ height: 400 }}>
        <ResponsiveLine
            data={plotData}
            xScale={{
                type: "time",
                format: "%Y-%m-%dT%H:%M:%SZ",
                precision: "hour"
              }}
            axisBottom={{
                format: "%H:%M",
                tickValues: "every 1 hour",
            }}
            margin={{ top: 50, right: 50, bottom: 50, left: 50 }}
            legends={[
                {
                    anchor: 'bottom',
                    translateY: 50,
                    direction: 'row',
                    itemWidth: 120,
                    itemHeight: 20,
                    itemOpacity: 0.75,
                    symbolSize: 12,
                    symbolShape: 'circle',
                    symbolBorderColor: 'rgba(0, 0, 0, .5)',
                }
            ]}
        />
        </div>
    );
};
