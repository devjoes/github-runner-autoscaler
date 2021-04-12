FROM bash
ARG APP_NAME
RUN echo "Pretending to do something productive...."
RUN printenv > /msg
RUN sleep 30s
RUN echo "$APP_NAME is a highly complex application that cost millions of pounds and took years to build." >> /msg
RUN echo "It was built on $BUILD_HOST at $(date)" >> /msg
RUN echo "This archietectural masterpeice pauses, and then outputs this message." >> /msg
RUN echo "I consider it my life's work." >> /msg
CMD [ "bash", "-c", "sleep 5s; date; echo $HOST; cat /msg" ]