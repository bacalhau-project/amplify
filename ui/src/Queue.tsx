import LaunchIcon from '@mui/icons-material/Launch';
import Chip from '@mui/material/Chip';
import { ArrayField, Create, Datagrid, DateField, List, Show, SimpleForm, SimpleShowLayout, TextField, TextInput, useRecordContext, FunctionField, SingleFieldList, ChipField, UrlField } from "react-admin";

export const QueueList = () => (
    <List title="Amplify Queue">
        <Datagrid rowClick="show"  >
            <TextField source="id" />
            <TextField source="meta.status" />
            <DateField source="meta.submitted" showTime={true} />
            <DateField source="meta.started" showTime={true} />
            <DateField source="meta.ended" showTime={true} />
        </Datagrid>
    </List>
);

const CustomChipField = (props) => {
    const { className, source, emptyText, ...rest } = props;
    const record = useRecordContext(props);
    const value = record && record[source];
    if (!value) return null;
    return (
        <div>
        <Chip
            key={value}
            label={value}
            icon={<LaunchIcon />}
            component="a"
            href={"https://ipfs.io/ipfs/" + value}
            target="_blank"
            variant="outlined"
            clickable
        />
        </div>
    );
};

export const QueueItem = () => (
    <Show >
        <SimpleShowLayout>
            <TextField source="id" />
            <ArrayField source="attributes.inputs">
                <SingleFieldList linkType={false}>
                    <CustomChipField
                        source="cid"
                    />
                </SingleFieldList>
            </ArrayField>
            <ArrayField source="attributes.outputs">
                <SingleFieldList linkType={false}>
                    <CustomChipField
                        source="cid"
                    />
                </SingleFieldList>
            </ArrayField>
            <DateField source="meta.submitted" showTime />
            <DateField source="meta.started" showTime />
            <DateField source="meta.ended" showTime />
            <TextField source="meta.status" />
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