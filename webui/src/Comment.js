import React from "react";
import gql from "graphql-tag";
import { withStyles } from "@material-ui/core/styles";

import Avatar from "@material-ui/core/Avatar";
import Card from "@material-ui/core/Card";
import CardContent from "@material-ui/core/CardContent";
import CardHeader from "@material-ui/core/CardHeader";
import Typography from "@material-ui/core/Typography";

const styles = theme => ({
  comment: {
    marginBottom: theme.spacing.unit
  }
});

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

Comment.fragment = gql`
  fragment Comment on Comment {
    message
    author {
      name
      email
    }
  }
`;

export default withStyles(styles)(Comment);
