import React from "react";
import { Query } from "react-apollo";
import gql from "graphql-tag";
import { withStyles } from "@material-ui/core/styles";

import Avatar from "@material-ui/core/Avatar";
import Card from "@material-ui/core/Card";
import CardContent from "@material-ui/core/CardContent";
import CardHeader from "@material-ui/core/CardHeader";
import Chip from "@material-ui/core/Chip";
import CircularProgress from "@material-ui/core/CircularProgress";
import Typography from "@material-ui/core/Typography";

const styles = theme => ({
  main: {
    maxWidth: 600,
    margin: "auto",
    marginTop: theme.spacing.unit * 4
  },
  labelList: {
    display: "flex",
    flexWrap: "wrap",
    marginTop: theme.spacing.unit
  },
  label: {
    marginRight: theme.spacing.unit
  },
  summary: {
    marginBottom: theme.spacing.unit * 2
  },
  comment: {
    marginBottom: theme.spacing.unit
  }
});

const QUERY = gql`
  query GetBug($id: BugID!) {
    bug(id: $id) {
      id
      title
      status
      labels
      comments {
        message
        author {
          name
          email
        }
      }
    }
  }
`;

const Comment = withStyles(styles)(({ comment, classes }) => (
  <Card className={classes.comment}>
    <CardHeader
      avatar={
        <Avatar aria-label={comment.author.name}>
          {comment.author.name[0].toUpperCase()}
        </Avatar>
      }
      title={comment.author.name}
      subheader={comment.author.email}
    />
    <CardContent>
      <Typography component="p">{comment.message}</Typography>
    </CardContent>
  </Card>
));

const BugView = withStyles(styles)(({ bug, classes }) => (
  <main className={classes.main}>
    <Card className={classes.summary}>
      <CardContent>
        <Typography variant="headline" component="h2">
          {bug.title}
        </Typography>
        <Typography variant="subheading" component="h3" title={bug.id}>
          #{bug.id.slice(0, 8)} • {bug.status.toUpperCase()}
        </Typography>
        <div className={classes.labelList}>
          {bug.labels.map(label => (
            <Chip key={label} label={label} className={classes.label} />
          ))}
        </div>
      </CardContent>
    </Card>

    {bug.comments.map((comment, index) => (
      <Comment key={index} comment={comment} />
    ))}
  </main>
));

const Bug = ({ match }) => (
  <Query query={QUERY} variables={{ id: match.params.id }}>
    {({ loading, error, data }) => {
      if (loading) return <CircularProgress />;
      if (error) return <p>Error.</p>;
      return <BugView bug={data.bug} />;
    }}
  </Query>
);

export default Bug;
