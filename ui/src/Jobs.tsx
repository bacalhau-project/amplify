import { Datagrid, List, TextField } from "react-admin";


export const JobList = () => (
    <List>
        <Datagrid rowClick="show">
            <TextField source="id" />
            <TextField source="image" />
            <TextField source="entrypoint" />
        </Datagrid>
    </List>
);