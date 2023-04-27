import LaunchIcon from '@mui/icons-material/Launch';
import Chip from '@mui/material/Chip';
import { ArrayField, Create, Datagrid, DateField, List, Show, SimpleForm, SimpleShowLayout, TextField, TextInput, useRecordContext, FunctionField, SingleFieldList, ChipField, UrlField } from "react-admin";

export const QueueList = () => (
    <List title="Amplify Queue" sort={{ field: 'meta.submitted', order: 'DESC' }}>
        <Datagrid rowClick="show" >
            <TextField source="id" sortable={false} />
            <TextField source="meta.status" sortable={false} />
            <DateField source="meta.submitted" showTime={true} />
            <DateField source="meta.started" showTime={true} sortable={false} />
            <DateField source="meta.ended" showTime={true} sortable={false} />
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
            href={"https://gateway.pinata.cloud/ipfs/" + value}
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