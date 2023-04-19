import LaunchIcon from '@mui/icons-material/Launch';
import Chip from '@mui/material/Chip';
import { ArrayField, Create, Datagrid, DateField, List, Show, SimpleForm, SimpleShowLayout, TextField, TextInput, useRecordContext } from "react-admin";

export const QueueList = () => (
    <List title="Amplify Queue">
        <Datagrid rowClick="show"  >
            <TextField source="id" />
            <TextField source="metadata.submitted" />
            <TextField source="metadata.status" />
        </Datagrid>
    </List>
);

const Inputs = () => {
    const record = useRecordContext();
    if (!record || !record.inputs) return null;
    return record.inputs.map((d) =>
        <Chip
            key={d.cid}
            label={d.cid}
            icon={<LaunchIcon />}
            component="a"
            href={"https://ipfs.io/ipfs/" + d.cid}
            target="_blank"
            variant="outlined"
            clickable
        />
    );
};


const Outputs = () => {
    const record = useRecordContext();
    if (!record || !record.outputs) return null;
    return record.outputs.map((d) =>
        <Chip
            key={d.cid}
            label={d.cid}
            icon={<LaunchIcon />}
            component="a"
            href={"https://ipfs.io/ipfs/" + d.cid}
            target="_blank"
            variant="outlined"
            clickable
        />
    );
};

export const QueueItem = () => (
    <Show >
        <SimpleShowLayout>
            <TextField source="id" />
            <ArrayField source="inputs">
                <Inputs />
            </ArrayField>
            <ArrayField source="outputs">
                <Outputs />
            </ArrayField>
            <DateField source="metadata.submitted" showTime />
            <DateField source="metadata.started" showTime />
            <DateField source="metadata.ended" showTime />
        </SimpleShowLayout>
    </Show >
);

export const QueueCreate = () => (
    <Create>
        <SimpleForm>
            <TextInput source="cid" />
        </SimpleForm>
    </Create>
);